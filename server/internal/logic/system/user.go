// Package system 系统管理-用户业务逻辑。
package system

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/logic/pwdpolicy"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/internal/storage"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/demox"
	"hinay.cn/admin/utility/excelx"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sUser struct{}

func NewUser() *sUser {
	return &sUser{}
}

func init() {
	service.RegisterUser(NewUser())
}

// List 分页列表。
func (s *sUser) List(ctx context.Context, req *v1.UserListReq) (res *v1.UserListRes, err error) {
	q := dao.SysUser.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		// 括号分组: 避免关键词 OR 条件逃逸出 deleted_at 过滤 (WhereOr 顶层分组陷阱)
		q = q.Where("(username LIKE ? OR nickname LIKE ?)", kw, kw)
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	// 组织数据权限: 按当前用户角色的数据范围过滤可见用户 (admin 不受限)
	q, derr := service.DataScope().Apply(ctx, q, "org_id", "id")
	if derr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, derr, "数据权限计算失败")
	}
	total, err := q.Ctx(ctx).Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.SysUser
	if err = q.Ctx(ctx).
		Page(req.Page, req.PageSize).
		Order("id DESC").
		Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 从 Casbin 加载角色 (g 行按用户ID关联; code 仅为展示标识)
	userRoleIds := make(map[uint64][]uint64, len(rows))
	roleIdSet := make(map[uint64]struct{})
	for _, u := range rows {
		ids, _ := casbinx.GetUserRoles(ctx, u.Id)
		userRoleIds[u.Id] = ids
		for _, rid := range ids {
			roleIdSet[rid] = struct{}{}
		}
	}
	distinctIds := make([]uint64, 0, len(roleIdSet))
	for rid := range roleIdSet {
		distinctIds = append(distinctIds, rid)
	}
	roleCodeMap := loadRoleCodeMap(ctx, distinctIds)

	list := make([]*v1.UserVO, 0, len(rows))
	for _, u := range rows {
		ids := userRoleIds[u.Id]
		codes := make([]string, 0, len(ids))
		for _, rid := range ids {
			if c, ok := roleCodeMap[rid]; ok {
				codes = append(codes, c)
			}
		}
		orgName := getOrgName(ctx, u.OrgId)
		list = append(list, &v1.UserVO{
			Id:        u.Id,
			Username:  u.Username,
			Nickname:  u.Nickname,
			Avatar:    storage.ViewURL(ctx, u.Avatar),
			Email:     u.Email,
			Phone:     u.Phone,
			OrgId:     u.OrgId,
			OrgName:   orgName,
			Status:    u.Status,
			Remark:    u.Remark,
			RoleIds:   ids,
			Roles:     codes,
			CreatedAt: u.CreatedAt,
		})
	}
	page := response.Page(list, int64(total), req.Page, req.PageSize)
	r := v1.UserListRes(page)
	return &r, nil
}

// Detail 详情。
func (s *sUser) Detail(ctx context.Context, req *v1.UserDetailReq) (res *v1.UserDetailRes, err error) {
	var u *model.SysUser
	if err = dao.SysUser.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&u); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if u == nil {
		return nil, xerror.New(xerror.CodeUserNotFound)
	}
	roleIds, _ := casbinx.GetUserRoles(ctx, u.Id)
	roleCodeMap := loadRoleCodeMap(ctx, roleIds)
	codes := make([]string, 0, len(roleIds))
	for _, rid := range roleIds {
		if c := roleCodeMap[rid]; c != "" {
			codes = append(codes, c)
		}
	}
	orgName := getOrgName(ctx, u.OrgId)
	return &v1.UserDetailRes{UserVO: &v1.UserVO{
		Id:        u.Id,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    storage.ViewURL(ctx, u.Avatar),
		Email:     u.Email,
		Phone:     u.Phone,
		OrgId:     u.OrgId,
		OrgName:   orgName,
		Status:    u.Status,
		Remark:    u.Remark,
		RoleIds:   roleIds,
		Roles:     codes,
		CreatedAt: u.CreatedAt,
	}}, nil
}

// Create 新增。
func (s *sUser) Create(ctx context.Context, req *v1.UserCreateReq) (res *v1.UserCreateRes, err error) {
	cnt, _ := dao.SysUser.Ctx(ctx).Where("username", req.Username).Where("deleted_at IS NULL").Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeUsernameExists)
	}
	// 密码由管理员设置: 须满足密码策略, 且标记用户下次登录强制改密
	if perr := pwdpolicy.Validate(ctx, req.Password); perr != nil {
		return nil, perr
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if req.Status == 0 {
		req.Status = consts.StatusEnabled
	}
	id, err := dao.SysUser.Ctx(ctx).Data(g.Map{
		"username":        req.Username,
		"password":        hash,
		"nickname":        req.Nickname,
		"email":           req.Email,
		"phone":           req.Phone,
		"org_id":          req.OrgId,
		"status":          req.Status,
		"remark":          req.Remark,
		"pwd_updated_at":  gtime.Now(),
		"must_change_pwd": 1,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 分配角色: g 策略直接以 用户ID/角色ID 关联。
	// 任一环节失败都需要回滚已创建的用户行, 避免出现"用户已建但无角色"的脏数据。
	if len(req.RoleIds) > 0 {
		roleIds, rerr := existingRoleIds(ctx, req.RoleIds)
		if rerr != nil {
			g.Log().Errorf(ctx, "UserCreate existingRoleIds failed, roleIds=%v err=%v", req.RoleIds, rerr)
			rollbackUser(ctx, id)
			return nil, xerror.Wrap(xerror.CodeBusinessError, rerr)
		}
		if len(roleIds) != len(req.RoleIds) {
			g.Log().Warningf(ctx, "UserCreate role mismatch, roleIds=%v exists=%v", req.RoleIds, roleIds)
		}
		if len(roleIds) == 0 {
			rollbackUser(ctx, id)
			return nil, xerror.New(xerror.CodeBusinessError, "指定的角色不存在或已被删除")
		}
		if serr := casbinx.SetUserRoles(ctx, uint64(id), roleIds); serr != nil {
			g.Log().Errorf(ctx, "UserCreate SetUserRoles failed, userId=%d roleIds=%v err=%v", id, roleIds, serr)
			rollbackUser(ctx, id)
			return nil, xerror.Wrap(xerror.CodeBusinessError, serr)
		}
	}
	return &v1.UserCreateRes{Id: uint64(id)}, nil
}

// rollbackUser 物理删除刚插入的用户, 用于角色绑定失败时的补偿。
func rollbackUser(ctx context.Context, id int64) {
	if _, err := dao.SysUser.Ctx(ctx).Where("id", id).Delete(); err != nil {
		g.Log().Errorf(ctx, "rollbackUser failed, id=%d err=%v", id, err)
	}
}

// Update 修改。
func (s *sUser) Update(ctx context.Context, req *v1.UserUpdateReq) (res *v1.UserUpdateRes, err error) {
	// 角色解析: 过滤掉已删除的角色, 与创建口径一致
	var roleIds []uint64
	if req.RoleIds != nil {
		roleIds, err = existingRoleIds(ctx, req.RoleIds)
		if err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
	}
	// 内置 admin (id=1) 保护: 不允许禁用, 不允许解绑超管角色 (内置角色 id=1)
	if req.Id == 1 {
		if req.Status != consts.StatusEnabled {
			return nil, xerror.New(xerror.CodeBusinessError, "内置管理员不可禁用")
		}
		if req.RoleIds != nil && !slices.Contains(roleIds, consts.RoleAdminId) {
			return nil, xerror.New(xerror.CodeBusinessError, "内置管理员不可解绑超管角色")
		}
	}
	if _, err = dao.SysUser.Ctx(ctx).
		Where("id", req.Id).Where("deleted_at IS NULL").Ctx(ctx).
		Data(g.Map{
			"nickname": req.Nickname,
			"email":    req.Email,
			"phone":    req.Phone,
			"org_id":   req.OrgId,
			"status":   req.Status,
			"remark":   req.Remark,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if req.RoleIds != nil {
		if serr := casbinx.SetUserRoles(ctx, req.Id, roleIds); serr != nil {
			g.Log().Errorf(ctx, "UserUpdate SetUserRoles failed, userId=%d roleIds=%v err=%v", req.Id, roleIds, serr)
			return nil, xerror.Wrap(xerror.CodeBusinessError, serr)
		}
	}
	return &v1.UserUpdateRes{}, nil
}

// Delete 软删除 + 清理 Casbin g 策略。
func (s *sUser) Delete(ctx context.Context, req *v1.UserDeleteReq) (res *v1.UserDeleteRes, err error) {
	if req.Id == 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "内置管理员不可删除")
	}

	if _, err = dao.SysUser.Ctx(ctx).Where("id", req.Id).
		Data(g.Map{"deleted_at": gtime.Now()}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 清理 Casbin g 策略中该用户的角色映射 (按用户ID)
	_ = casbinx.SetUserRoles(ctx, req.Id, nil)
	return &v1.UserDeleteRes{}, nil
}

// ResetPwd 重置密码。
// 演示环境 (demo.enable=true) 下全局禁止。
func (s *sUser) ResetPwd(ctx context.Context, req *v1.UserResetPwdReq) (res *v1.UserResetPwdRes, err error) {
	if gerr := demox.Guard(ctx); gerr != nil {
		return nil, gerr
	}
	// 重置的密码须满足密码策略, 且该用户下次登录强制改密
	if perr := pwdpolicy.Validate(ctx, req.Password); perr != nil {
		return nil, perr
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if _, err = dao.SysUser.Ctx(ctx).Where("id", req.Id).
		Data(g.Map{
			"password":        hash,
			"pwd_updated_at":  gtime.Now(),
			"must_change_pwd": 1,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.UserResetPwdRes{}, nil
}

// existingRoleIds 过滤出仍然存在的角色ID (软删角色视为不存在)。
func existingRoleIds(ctx context.Context, ids []uint64) ([]uint64, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	values, err := dao.SysRole.Ctx(ctx).
		WhereIn("id", ids).
		Where("deleted_at IS NULL").
		Fields("id").
		Ctx(ctx).Array()
	if err != nil {
		return nil, err
	}
	out := make([]uint64, 0, len(values))
	for _, v := range values {
		if id := v.Uint64(); id > 0 {
			out = append(out, id)
		}
	}
	return out, nil
}

// loadRoleCodeMap 批量将角色ID解析为 code 映射 (角色已删除的不出现, 查询失败返回空映射)。
func loadRoleCodeMap(ctx context.Context, ids []uint64) map[uint64]string {
	m := make(map[uint64]string, len(ids))
	if len(ids) == 0 {
		return m
	}
	res, err := dao.SysRole.Ctx(ctx).
		WhereIn("id", ids).
		Where("deleted_at IS NULL").
		Fields("id, code").
		Ctx(ctx).All()
	if err != nil {
		return m
	}
	for _, r := range res {
		m[r["id"].Uint64()] = r["code"].String()
	}
	return m
}

// getOrgName 根据组织ID获取组织名称, 不存在或 orgId=0 时返回空串。
func getOrgName(ctx context.Context, orgId uint64) string {
	if orgId == 0 {
		return ""
	}
	val, err := dao.SysOrg.Ctx(ctx).
		Where("id", orgId).Where("deleted_at IS NULL").
		Fields("name").Value()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val.String())
}

// ---------------------------------------------------------------------------
// Excel 通用导入导出示例 (工具链见 utility/excelx)
// ---------------------------------------------------------------------------

// 用户导入模板/解析共用的表头 (导入按下标取列, 必须与模板一致)。
var userImportHeaders = []string{
	"账号*", "昵称*", "密码(留空默认123456)", "组织ID", "邮箱", "手机号", "备注",
}

// 用户导出行数上限: 防止全量导出拖垮内存, 更大数据量应走异步导出。
const userExportMaxRows = 10000

// 用户导入行数上限。
const userImportMaxRows = 1000

// userImportDefaultPassword 导入时密码留空的默认密码。
const userImportDefaultPassword = "123456"

// Export 用户列表导出 (通用导出示例)。
// 过滤条件与列表一致且叠加数据权限, 不分页; 内容直接写入响应流下载。
func (s *sUser) Export(ctx context.Context, req *v1.UserExportReq) (res *v1.UserExportRes, err error) {
	q := dao.SysUser.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("(username LIKE ? OR nickname LIKE ?)", kw, kw)
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	// 与列表一致: 叠加组织数据权限
	q, derr := service.DataScope().Apply(ctx, q, "org_id", "id")
	if derr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, derr, "数据权限计算失败")
	}
	var rows []*model.SysUser
	if err = q.Limit(userExportMaxRows).Order("id ASC").Ctx(ctx).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 组织名称批量查询 (避免逐行 N+1)
	orgIds := make([]uint64, 0, len(rows))
	for _, u := range rows {
		if u.OrgId > 0 {
			orgIds = append(orgIds, u.OrgId)
		}
	}
	orgNames := map[uint64]string{}
	if len(orgIds) > 0 {
		all, oerr := dao.SysOrg.Ctx(ctx).Fields("id, name").WhereIn("id", orgIds).Where("deleted_at IS NULL").All()
		if oerr == nil {
			for _, o := range all {
				orgNames[o["id"].Uint64()] = o["name"].String()
			}
		}
	}

	data := make([][]any, 0, len(rows))
	for _, u := range rows {
		status := "禁用"
		if u.Status == consts.StatusEnabled {
			status = "启用"
		}
		data = append(data, []any{
			u.Id, u.Username, u.Nickname, orgNames[u.OrgId],
			u.Email, u.Phone, status, u.Remark, u.CreatedAt,
		})
	}
	headers := []string{"ID", "账号", "昵称", "组织", "邮箱", "手机号", "状态", "备注", "创建时间"}

	content, berr := excelx.Build("用户列表", headers, data)
	if berr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, berr, "生成导出文件失败")
	}
	excelx.WriteToResponse(ctx, fmt.Sprintf("用户列表_%s.xlsx", gtime.Now().Format("YmdHis")), content)
	return &v1.UserExportRes{}, nil
}

// ImportTemplate 用户导入模板下载 (与 Import 的解析表头严格一致)。
func (s *sUser) ImportTemplate(ctx context.Context, req *v1.UserImportTemplateReq) (res *v1.UserImportTemplateRes, err error) {
	content, berr := excelx.Build("用户导入", userImportHeaders, nil)
	if berr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, berr, "生成模板失败")
	}
	excelx.WriteToResponse(ctx, "用户导入模板.xlsx", content)
	return &v1.UserImportTemplateRes{}, nil
}

// Import 用户导入 (通用导入示例): 逐行校验, 行级错误不中断整体导入。
func (s *sUser) Import(ctx context.Context, req *v1.UserImportReq) (res *v1.UserImportRes, err error) {
	// 审计: 暂存导入文件摘要, OperationLog 记录入库
	contextx.SetAuditUpload(ctx, fmt.Sprintf("%s (%d bytes, %s)",
		filepath.Base(req.File.Filename), req.File.Size, req.File.Header.Get("Content-Type")))
	f, oerr := req.File.Open()
	if oerr != nil {
		return nil, xerror.Wrap(xerror.CodeParamInvalid, oerr, "读取上传文件失败")
	}
	defer f.Close()
	content, rerr := io.ReadAll(f)
	if rerr != nil {
		return nil, xerror.Wrap(xerror.CodeParamInvalid, rerr, "读取上传文件失败")
	}
	rows, perr := excelx.Read(content)
	if perr != nil {
		return nil, xerror.New(xerror.CodeParamInvalid, "无法解析 xlsx 文件, 请使用下载的模板填写")
	}
	if len(rows) < 2 {
		return nil, xerror.New(xerror.CodeParamInvalid, "文件没有数据行")
	}

	// 表头一致性校验 (忽略尾随空列差异)
	for i, want := range userImportHeaders {
		if i >= len(rows[0]) || strings.TrimSpace(rows[0][i]) == "" {
			continue
		}
		if strings.TrimSuffix(rows[0][i], "*") != strings.TrimSuffix(want, "*") &&
			strings.TrimSpace(rows[0][i]) != want {
			return nil, xerror.New(xerror.CodeParamInvalid, fmt.Sprintf("表头第 %d 列应为「%s」", i+1, want))
		}
	}

	res = &v1.UserImportRes{Errors: []string{}}
	seen := make(map[string]struct{}) // 文件内查重
	cell := func(cols []string, i int) string {
		if i < len(cols) {
			return strings.TrimSpace(cols[i])
		}
		return ""
	}

	for i := 1; i < len(rows); i++ {
		if res.SuccessCount+res.FailCount >= userImportMaxRows {
			res.Errors = append(res.Errors, fmt.Sprintf("已达单次导入上限 %d 行, 其余未处理", userImportMaxRows))
			break
		}
		cols := rows[i]
		rowno := i + 1
		username, nickname := cell(cols, 0), cell(cols, 1)
		pass, orgIdStr := cell(cols, 2), cell(cols, 3)
		email, phone, remark := cell(cols, 4), cell(cols, 5), cell(cols, 6)

		// 整行为空跳过
		if username == "" && nickname == "" && email == "" && phone == "" {
			continue
		}

		fail := func(reason string) {
			res.FailCount++
			if len(res.Errors) < 50 {
				res.Errors = append(res.Errors, fmt.Sprintf("第 %d 行: %s", rowno, reason))
			}
		}

		if l := len([]rune(username)); l < 2 || l > 32 {
			fail("账号长度需 2-32 位")
			continue
		}
		if nickname == "" {
			fail("昵称不能为空")
			continue
		}
		if _, dup := seen[username]; dup {
			fail("账号在文件内重复")
			continue
		}
		if cnt, _ := dao.SysUser.Ctx(ctx).Where("username", username).Where("deleted_at IS NULL").Count(); cnt > 0 {
			fail("账号已存在")
			continue
		}
		if pass == "" {
			pass = userImportDefaultPassword
		}
		// 导入密码 (含默认密码) 须满足当前密码策略
		if perr := pwdpolicy.Validate(ctx, pass); perr != nil {
			fail(perr.Error())
			continue
		}
		var orgId uint64
		if orgIdStr != "" {
			v, cerr := strconv.ParseUint(orgIdStr, 10, 64)
			if cerr != nil {
				fail("组织ID必须是数字")
				continue
			}
			if cnt, _ := dao.SysOrg.Ctx(ctx).Where("id", v).Where("deleted_at IS NULL").Count(); cnt == 0 {
				fail("组织ID不存在")
				continue
			}
			orgId = v
		}
		if email != "" && !strings.Contains(email, "@") {
			fail("邮箱格式不正确")
			continue
		}

		hash, herr := password.Hash(pass)
		if herr != nil {
			fail("密码加密失败")
			continue
		}
		if _, ierr := dao.SysUser.Ctx(ctx).Data(g.Map{
			"username":        username,
			"password":        hash,
			"nickname":        nickname,
			"org_id":          orgId,
			"email":           email,
			"phone":           phone,
			"remark":          remark,
			"status":          consts.StatusEnabled,
			"pwd_updated_at":  gtime.Now(),
			"must_change_pwd": 1,
		}).Insert(); ierr != nil {
			fail("写入失败: " + ierr.Error())
			continue
		}
		seen[username] = struct{}{}
		res.SuccessCount++
	}
	return res, nil
}
