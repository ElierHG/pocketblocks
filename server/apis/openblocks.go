package apis

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pedrozadotdev/pocketblocks/server/daos"
	"github.com/pedrozadotdev/pocketblocks/server/forms"
	"github.com/pedrozadotdev/pocketblocks/server/models"
	"github.com/pedrozadotdev/pocketblocks/server/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	pbModels "github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tokens"
)

const cookieName = "pb_auth"

type openblocksApi struct {
	app *pocketbase.PocketBase
	dao *daos.Dao
}

func BindOpenblocksApi(app *pocketbase.PocketBase, dao *daos.Dao, e *echo.Echo) {
	api := &openblocksApi{app: app, dao: dao}

	// Auth
	e.POST("/api/auth/form/login", api.authLogin)
	e.POST("/api/auth/logout", api.authLogout)
	e.POST("/api/auth/email/bind", api.authEmailBind)

	// Users
	e.GET("/api/v1/users/me", api.usersMe)
	e.PUT("/api/v1/users", api.usersUpdate)
	e.GET("/api/users/currentUser", api.usersCurrentUser)
	e.PUT("/api/v1/users/password", api.usersPassword)
	e.PUT("/api/users/mark-status", api.usersMarkStatus)

	// Applications
	e.GET("/api/v1/applications/home", api.applicationsHome)
	e.GET("/api/v1/applications/:slug/view", api.applicationView)
	e.GET("/api/v1/applications/:slug/permissions", api.applicationPermissionsGet)
	e.PUT("/api/v1/applications/:slug/permissions", api.applicationPermissionsUpdate)
	e.DELETE("/api/v1/applications/:slug/permissions/:permId", api.applicationPermissionsDelete)
	e.POST("/api/v1/applications/:slug/publish", api.applicationPublish)
	e.GET("/api/v1/applications/:slug", api.applicationView)
	e.POST("/api/v1/applications", api.applicationCreate)
	e.PUT("/api/v1/applications/:slug", api.applicationUpdate)
	e.DELETE("/api/v1/applications/:slug", api.applicationDelete)
	e.GET("/api/applications/list", api.applicationsList)
	e.PUT("/api/applications/recycle/:slug", api.applicationRecycle)
	e.PUT("/api/applications/restore/:slug", api.applicationRestore)
	e.GET("/api/applications/recycle/list", api.applicationsRecycleList)
	e.PUT("/api/applications/:slug/public-to-all", api.applicationPublicToAll)

	// Folders
	e.GET("/api/folders/elements", api.foldersElements)
	e.POST("/api/folders", api.foldersCreate)
	e.PUT("/api/folders", api.foldersUpdate)
	e.PUT("/api/folders/move/:appSlug", api.foldersMove)
	e.DELETE("/api/folders/:id", api.foldersDelete)

	// Groups
	e.GET("/api/v1/groups/list", api.groupsList)

	// Snapshots
	e.GET("/api/application/history-snapshots/:appSlug/:id", api.snapshotView)
	e.GET("/api/application/history-snapshots/:appSlug", api.snapshotList)
	e.POST("/api/application/history-snapshots", api.snapshotCreate)

	// Configs
	e.GET("/api/v1/configs", api.configsView)
	e.PUT("/api/v1/configs/custom-configs", api.configsUpdate)

	// Organizations
	e.GET("/api/organizations/:id/common-settings", api.orgCommonSettings)
	e.PUT("/api/organizations/:id/common-settings", api.orgCommonSettingsUpdate)
	e.GET("/api/v1/organizations/:id/members", api.orgMembers)

	// Constants (empty responses)
	emptyList := func(c echo.Context) error { return okResp(c, []interface{}{}) }
	e.GET("/api/misc/js-library/recommendations", emptyList)
	e.GET("/api/misc/js-library/metas", emptyList)
	e.GET("/api/v1/organizations/:orgId/datasourceTypes", emptyList)
	e.GET("/api/v1/datasources/listByApp", emptyList)
	e.GET("/api/library-queries/dropDownList", emptyList)
	e.GET("/api/v1/datasources/jsDatasourcePlugins", emptyList)

	// Avatar upload
	e.POST("/api/users/avatar", api.usersAvatarUpload)
}

// --- Response helpers ---

func okResp(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": 1, "message": "", "success": true, "data": data,
	})
}

func errResp(c echo.Context, status int, msg string) error {
	return c.JSON(status, map[string]interface{}{
		"code": status, "message": msg, "success": false,
	})
}

// --- Auth helpers ---

func (api *openblocksApi) getAuthToken(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}
	cookie, err := c.Cookie(cookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func (api *openblocksApi) getAdmin(c echo.Context) *pbModels.Admin {
	token := api.getAuthToken(c)
	if token == "" {
		return nil
	}
	admin, err := api.app.Dao().FindAdminByToken(token, api.app.Settings().AdminAuthToken.Secret)
	if err != nil {
		return nil
	}
	return admin
}

func (api *openblocksApi) getAuthRecord(c echo.Context) *pbModels.Record {
	token := api.getAuthToken(c)
	if token == "" {
		return nil
	}
	record, err := api.app.Dao().FindAuthRecordByToken(token, api.app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return nil
	}
	return record
}

func (api *openblocksApi) isAdmin(c echo.Context) bool {
	return api.getAdmin(c) != nil
}

func (api *openblocksApi) isLoggedIn(c echo.Context) bool {
	return api.isAdmin(c) || api.getAuthRecord(c) != nil
}

func (api *openblocksApi) requireAuth(c echo.Context) error {
	if !api.isLoggedIn(c) {
		return errResp(c, 401, "Unauthorized")
	}
	return nil
}

func (api *openblocksApi) requireAdmin(c echo.Context) error {
	if !api.isAdmin(c) {
		return errResp(c, 401, "Unauthorized")
	}
	return nil
}

func setAuthCookie(c echo.Context, token string) {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 30, // 30 days
	}
	c.SetCookie(cookie)
}

func clearAuthCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
}

// --- Auth config builder ---

type oauthInfo struct {
	Name           string `json:"name"`
	CustomName     string `json:"customName"`
	CustomIconUrl  string `json:"customIconUrl"`
	DefaultName    string `json:"defaultName"`
	DefaultIconUrl string `json:"defaultIconUrl"`
}

func (api *openblocksApi) buildAuthConfigs() []interface{} {
	store := api.dao.GetPblStore()
	settings := api.dao.GetPblSettings()

	userFieldUpdate := store.Get(utils.UserFieldUpdateKey).([]string)
	authMethods := store.Get(utils.UserAuthsKey).([]string)
	canUserSignUp := store.Get(utils.CanUserSignUpKey).(bool)
	setupFirstAdmin := store.Get(utils.SetupFirstAdminKey).(bool)
	smtpStatus := store.Get(utils.SmtpStatusKey).(bool)
	localAuthInfo := store.Get(utils.LocalAuthGeneralInfoKey).(utils.LocalAuthGeneralInfo)

	settingsClone, _ := settings.Clone()

	types := []string{}
	if slices.Contains(authMethods, "email") {
		types = append(types, "email")
	}
	if slices.Contains(authMethods, "username") {
		types = append(types, "username")
	}

	oauthList := []oauthInfo{}
	for _, m := range authMethods {
		if m == "email" || m == "username" {
			continue
		}
		oa := settingsClone.GetOauthByAuthName(m)
		oauthList = append(oauthList, oauthInfo{
			Name:           m,
			CustomName:     oa.CustomName,
			CustomIconUrl:  oa.CustomIconUrl,
			DefaultName:    strings.ToUpper(m[:1]) + m[1:],
			DefaultIconUrl: "/_/images/oauth2/" + m + ".svg",
		})
	}

	return []interface{}{map[string]interface{}{
		"authType":       "FORM",
		"id":             "EMAIL",
		"enable":         slices.Contains(authMethods, "email") || slices.Contains(authMethods, "username"),
		"enableRegister": canUserSignUp,
		"source":         "EMAIL",
		"sourceName":     "EMAIL",
		"customProps": map[string]interface{}{
			"label":         settingsClone.Auths.Local.Label,
			"mask":          settingsClone.Auths.Local.IdInputMask,
			"type":          types,
			"allowUpdate":   userFieldUpdate,
			"setupAdmin":    setupFirstAdmin,
			"smtp":          smtpStatus,
			"localAuthInfo": localAuthInfo,
		},
		"oauth": oauthList,
	}}
}

// --- Auth routes ---

func (api *openblocksApi) authLogin(c echo.Context) error {
	var body struct {
		LoginId    string `json:"loginId"`
		Password   string `json:"password"`
		Register   bool   `json:"register"`
		Source     string `json:"source"`
		AuthId     string `json:"authId"`
		ResetToken string `json:"resetToken"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.AuthId == "RESET_PASSWORD" {
		if body.ResetToken != "" {
			_, err := api.app.Dao().FindAuthRecordByToken(body.ResetToken, api.app.Settings().RecordAuthToken.Secret)
			if err != nil {
				return errResp(c, 400, "Invalid or expired token")
			}
			return okResp(c, nil)
		}
		record, err := api.app.Dao().FindAuthRecordByEmail("users", body.LoginId)
		if err != nil || record == nil {
			return errResp(c, 400, "User not found")
		}
		return okResp(c, nil)
	}

	if body.Register {
		return api.handleSignup(c, body.LoginId, body.Password)
	}

	return api.handleLogin(c, body.LoginId, body.Password)
}

func (api *openblocksApi) handleLogin(c echo.Context, loginId, password string) error {
	// Try admin auth first
	admin, err := api.app.Dao().FindAdminByEmail(loginId)
	if err == nil && admin.ValidatePassword(password) {
		token, err := tokens.NewAdminAuthToken(api.app, admin)
		if err != nil {
			return errResp(c, 500, "Failed to generate token")
		}
		setAuthCookie(c, token)
		return okResp(c, nil)
	}

	// Try user auth by email
	record, err := api.app.Dao().FindAuthRecordByEmail("users", loginId)
	if err == nil && record.ValidatePassword(password) {
		token, err := tokens.NewRecordAuthToken(api.app, record)
		if err != nil {
			return errResp(c, 500, "Failed to generate token")
		}
		setAuthCookie(c, token)
		return okResp(c, nil)
	}

	// Try user auth by username
	record, err = api.app.Dao().FindAuthRecordByUsername("users", loginId)
	if err == nil && record.ValidatePassword(password) {
		token, err := tokens.NewRecordAuthToken(api.app, record)
		if err != nil {
			return errResp(c, 500, "Failed to generate token")
		}
		setAuthCookie(c, token)
		return okResp(c, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": 5608, "message": "Invalid email/username or password.", "success": false,
	})
}

func (api *openblocksApi) handleSignup(c echo.Context, loginId, password string) error {
	parts := strings.Split(loginId, "\n")
	email := ""
	username := ""
	name := ""
	if len(parts) >= 1 {
		email = parts[0]
	}
	if len(parts) >= 2 {
		username = parts[1]
	}
	if len(parts) >= 3 {
		name = parts[2]
	}

	store := api.dao.GetPblStore()
	setupFirstAdmin := store.Get(utils.SetupFirstAdminKey).(bool)

	if setupFirstAdmin {
		admin := &pbModels.Admin{}
		admin.Email = email
		admin.SetPassword(password)
		if err := api.app.Dao().SaveAdmin(admin); err != nil {
			return errResp(c, 400, err.Error())
		}
		token, err := tokens.NewAdminAuthToken(api.app, admin)
		if err != nil {
			return errResp(c, 500, "Failed to generate token")
		}
		setAuthCookie(c, token)
		return okResp(c, nil)
	}

	collection, err := api.app.Dao().FindCollectionByNameOrId("users")
	if err != nil {
		return errResp(c, 500, "Users collection not found")
	}
	record := pbModels.NewRecord(collection)
	record.Set("email", email)
	record.Set("username", username)
	record.Set("name", name)
	record.SetPassword(password)
	if err := api.app.Dao().SaveRecord(record); err != nil {
		return errResp(c, 401, err.Error())
	}

	token, err := tokens.NewRecordAuthToken(api.app, record)
	if err != nil {
		return errResp(c, 500, "Failed to generate token")
	}
	setAuthCookie(c, token)
	return okResp(c, nil)
}

func (api *openblocksApi) authLogout(c echo.Context) error {
	clearAuthCookie(c)
	return okResp(c, nil)
}

func (api *openblocksApi) authEmailBind(c echo.Context) error {
	var body struct {
		Email    string `json:"email"`
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.Email != "" {
		record := api.getAuthRecord(c)
		if record == nil {
			return errResp(c, 401, "Unauthorized")
		}
		return okResp(c, nil)
	}
	return okResp(c, nil)
}

// --- User routes ---

func (api *openblocksApi) getUserAvatarUrl(record *pbModels.Record) string {
	avatar := record.GetString("avatar")
	if avatar != "" {
		return "/api/files/users/" + record.Id + "/" + avatar + "?thumb=100x100"
	}
	return ""
}

func (api *openblocksApi) usersMe(c echo.Context) error {
	admin := api.getAdmin(c)
	authRecord := api.getAuthRecord(c)

	if admin == nil && authRecord == nil {
		return okResp(c, map[string]interface{}{
			"id": nil, "orgAndRoles": nil, "currentOrgId": nil,
			"username": "anonymous", "connections": nil,
			"avatar": nil, "avatarUrl": nil, "hasPassword": false,
			"hasSetNickname": false, "hasShownNewUserGuidance": false,
			"userStatus": nil, "createdTimeMs": 0, "ip": "",
			"enabled": false, "anonymous": true, "orgDev": false,
			"isAnonymous": true, "isEnabled": false,
		})
	}

	settings, err := api.dao.GetPblSettings().Clone()
	if err != nil {
		return errResp(c, 500, "Failed to load settings")
	}

	isAdm := admin != nil
	var userId, userName, userEmail, avatarUrl string
	var createdTimeMs int64
	var showTutorialContainsUser bool

	if isAdm {
		userId = admin.Id
		userName = "Admin"
		userEmail = admin.Email
		avatarUrl = "/_/images/avatars/avatar" + strconv.Itoa(admin.Avatar) + ".svg"
		createdTimeMs = admin.Created.Time().UnixMilli()
		showTutorialContainsUser = slices.Contains(settings.ShowTutorial, admin.Id)
	} else {
		userId = authRecord.Id
		name := authRecord.GetString("name")
		if name == "NONAME" {
			name = "Unknown"
		}
		userName = name
		userEmail = authRecord.Email()
		avatarUrl = api.getUserAvatarUrl(authRecord)
		createdTimeMs = authRecord.Created.Time().UnixMilli()
		showTutorialContainsUser = slices.Contains(settings.ShowTutorial, authRecord.Id)
	}

	role := "member"
	if isAdm {
		role = "admin"
	}

	themeList := settings.Themes
	libs := settings.Libs
	plugins := settings.Plugins

	commonSettings := map[string]interface{}{
		"themeList":          themeList,
		"defaultTheme":      settings.ThemeId,
		"preloadCSS":        settings.Css,
		"preloadJavaScript": settings.Script,
		"preloadLibs":       libs,
		"npmPlugins":        plugins,
	}
	if settings.HomePageAppSlug != "" {
		commonSettings["defaultHomePage"] = settings.HomePageAppSlug
	}

	connectionUsername := userName
	if isAdm {
		connectionUsername = "ADMIN"
	} else {
		connectionUsername = authRecord.Username()
	}

	return okResp(c, map[string]interface{}{
		"id": userId,
		"orgAndRoles": []interface{}{map[string]interface{}{
			"org": map[string]interface{}{
				"id":                          "ORG_ID",
				"createdBy":                   "",
				"name":                        settings.Name,
				"isAutoGeneratedOrganization": true,
				"contactName":                 nil,
				"contactEmail":                nil,
				"contactPhoneNumber":          nil,
				"source":                      nil,
				"thirdPartyCompanyId":         nil,
				"state":                       "ACTIVE",
				"commonSettings":              commonSettings,
				"logoUrl":                     settings.LogoUrl,
				"createTime":                  0,
				"authConfigs":                 api.buildAuthConfigs(),
			},
			"role": role,
		}},
		"currentOrgId": "ORG_ID",
		"username":     userName,
		"connections": []interface{}{map[string]interface{}{
			"authId": "EMAIL",
			"source": "EMAIL",
			"name":   userEmail,
			"avatar": avatarUrl,
			"rawUserInfo": map[string]interface{}{
				"email":    userEmail,
				"username": connectionUsername,
			},
			"tokens": []interface{}{},
		}},
		"avatar":               avatarUrl,
		"avatarUrl":            avatarUrl,
		"hasPassword":          true,
		"hasSetNickname":       true,
		"hasShownNewUserGuidance": false,
		"userStatus": map[string]interface{}{
			"newUserGuidance": !showTutorialContainsUser,
		},
		"createdTimeMs": createdTimeMs,
		"ip":            "",
		"enabled":       false,
		"anonymous":     false,
		"orgDev":        isAdm,
		"isAnonymous":   false,
		"isEnabled":     false,
	})
}

func (api *openblocksApi) usersUpdate(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	var body struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	record := api.getAuthRecord(c)
	if record == nil {
		return errResp(c, 401, "Unauthorized")
	}

	if body.Name != "" {
		record.Set("name", body.Name)
	}
	if body.Username != "" {
		record.Set("username", body.Username)
	}
	if err := api.app.Dao().SaveRecord(record); err != nil {
		return errResp(c, 400, err.Error())
	}

	return api.usersMe(c)
}

func (api *openblocksApi) usersCurrentUser(c echo.Context) error {
	admin := api.getAdmin(c)
	authRecord := api.getAuthRecord(c)

	if admin == nil && authRecord == nil {
		return okResp(c, map[string]interface{}{
			"id": "", "name": "ANONYMOUS", "avatarUrl": nil,
			"email": "", "ip": "", "groups": []interface{}{}, "extra": map[string]interface{}{},
		})
	}

	isAdm := admin != nil
	var userId, userName, email, avatar string

	if isAdm {
		userId = admin.Id
		userName = "Admin"
		email = admin.Email
		avatar = ""
	} else {
		userId = authRecord.Id
		name := authRecord.GetString("name")
		if name == "NONAME" {
			name = "Unknown"
		}
		userName = name
		email = authRecord.Email()
		avatar = api.getUserAvatarUrl(authRecord)
	}

	groups := []interface{}{}
	if !isAdm {
		records, err := api.app.Dao().FindRecordsByFilter(
			"groups",
			"users.id ?= \""+authRecord.Id+"\"",
			"-created", 500, 0,
		)
		if err == nil {
			for _, g := range records {
				groups = append(groups, map[string]interface{}{
					"groupId":   g.Id,
					"groupName": g.GetString("name"),
				})
			}
		}
	}

	return okResp(c, map[string]interface{}{
		"id": userId, "name": userName, "avatarUrl": avatar,
		"email": email, "ip": "", "groups": groups, "extra": map[string]interface{}{},
	})
}

func (api *openblocksApi) usersPassword(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	var body struct {
		NewPassword string `json:"newPassword"`
		OldPassword string `json:"oldPassword"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	admin := api.getAdmin(c)
	if admin != nil {
		admin.SetPassword(body.NewPassword)
		if err := api.app.Dao().SaveAdmin(admin); err != nil {
			return errResp(c, 400, err.Error())
		}
		token, _ := tokens.NewAdminAuthToken(api.app, admin)
		setAuthCookie(c, token)
		return okResp(c, nil)
	}

	record := api.getAuthRecord(c)
	if record == nil {
		return errResp(c, 401, "Unauthorized")
	}
	if !record.ValidatePassword(body.OldPassword) {
		return errResp(c, 403, "Invalid password!")
	}
	record.SetPassword(body.NewPassword)
	if err := api.app.Dao().SaveRecord(record); err != nil {
		return errResp(c, 400, err.Error())
	}
	token, _ := tokens.NewRecordAuthToken(api.app, record)
	setAuthCookie(c, token)
	return okResp(c, nil)
}

func (api *openblocksApi) usersMarkStatus(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	var body struct {
		Type string `json:"type"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.Type == "newUserGuidance" {
		admin := api.getAdmin(c)
		if admin != nil {
			api.dao.DeleteAdminFromPblSettingsTutorial(admin.Id)
		}
	}
	return okResp(c, nil)
}

func (api *openblocksApi) usersAvatarUpload(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	record := api.getAuthRecord(c)
	if record == nil {
		return errResp(c, 401, "Only users can upload avatars")
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return errResp(c, 400, "No file uploaded")
	}

	src, err := file.Open()
	if err != nil {
		return errResp(c, 500, "Failed to read file")
	}
	defer src.Close()

	record.Set("avatar", file)
	if err := api.app.Dao().SaveRecord(record); err != nil {
		return errResp(c, 500, err.Error())
	}
	return okResp(c, nil)
}

// --- Application helpers ---

func (api *openblocksApi) createAppListItem(app *models.Application, isAdm bool) map[string]interface{} {
	var appIconUrl interface{}
	var dsl map[string]interface{}
	if json.Unmarshal([]byte(app.AppDsl), &dsl) == nil {
		if s, ok := dsl["settings"].(map[string]interface{}); ok {
			if icon, ok := s["appIconUrl"].(string); ok && icon != "" {
				appIconUrl = icon
			}
		}
	}

	role := "viewer"
	if isAdm {
		role = "owner"
	}

	return map[string]interface{}{
		"orgId":             "ORG_ID",
		"applicationId":    app.Slug,
		"name":             app.Name,
		"createAt":         app.Created.Time().UnixMilli(),
		"role":             role,
		"applicationType":  app.Type,
		"applicationStatus": app.Status,
		"folderId":         app.FolderId,
		"lastViewTime":     app.Updated.Time().UnixMilli(),
		"lastModifyTime":   app.Updated.Time().UnixMilli(),
		"publicToAll":      app.Public,
		"folder":           false,
		"extra":            map[string]interface{}{"appIconUrl": appIconUrl},
	}
}

func (api *openblocksApi) getCorrectDSL(c echo.Context, app *models.Application) string {
	path := c.Request().Header.Get("Referer")
	if strings.Contains(path, "/edit") || strings.Contains(path, "/preview") {
		return app.EditDsl
	}
	return app.AppDsl
}

func (api *openblocksApi) createFullAppResponse(c echo.Context, app *models.Application) (map[string]interface{}, error) {
	isAdm := api.isAdmin(c)

	settings, err := api.dao.GetPblSettings().Clone()
	if err != nil {
		return nil, err
	}

	dslStr := api.getCorrectDSL(c, app)
	var dsl interface{}
	json.Unmarshal([]byte(dslStr), &dsl)

	commonSettings := map[string]interface{}{
		"themeList":          settings.Themes,
		"defaultTheme":      settings.ThemeId,
		"preloadCSS":        settings.Css,
		"preloadJavaScript": settings.Script,
		"preloadLibs":       settings.Libs,
		"npmPlugins":        settings.Plugins,
	}
	if settings.HomePageAppSlug != "" {
		commonSettings["defaultHomePage"] = settings.HomePageAppSlug
	}

	return map[string]interface{}{
		"applicationInfoView": api.createAppListItem(app, isAdm),
		"applicationDSL":     dsl,
		"moduleDSL":          map[string]interface{}{},
		"orgCommonSettings":  commonSettings,
		"templateId":         nil,
	}, nil
}

func (api *openblocksApi) listApps(c echo.Context, onlyRecycled bool, folderId string) ([]*models.Application, error) {
	query := api.dao.PblAppQuery().OrderBy("updated DESC", "created DESC")

	if onlyRecycled {
		query = query.AndWhere(dbx.HashExp{"status": "RECYCLED"})
	} else {
		query = query.AndWhere(dbx.NewExp("status != 'RECYCLED'"))
		if folderId != "" {
			query = query.AndWhere(dbx.HashExp{"folder": folderId})
		} else {
			query = query.AndWhere(dbx.Or(dbx.HashExp{"folder": ""}, dbx.HashExp{"folder": nil}))
		}
	}

	info := apis.RequestInfo(c)
	if info.Admin == nil && info.AuthRecord != nil {
		groups, _ := api.app.Dao().FindRecordsByFilter(
			"groups",
			"users.id ?= \""+info.AuthRecord.Id+"\"",
			"-created", 500, 0,
		)
		groupIds := []string{}
		for _, g := range groups {
			groupIds = append(groupIds, g.Id)
		}

		var filterExpr dbx.Expression = dbx.Or(
			dbx.HashExp{"public": true},
			dbx.HashExp{"allUsers": true},
		)
		if len(groupIds) > 0 {
			filterExpr = dbx.Or(
				filterExpr,
				dbx.Like("users", info.AuthRecord.Id),
				dbx.OrLike("groups", groupIds...),
			)
		} else {
			filterExpr = dbx.Or(
				filterExpr,
				dbx.Like("users", info.AuthRecord.Id),
			)
		}
		query = query.AndWhere(filterExpr)
	}

	apps := []*models.Application{}
	if err := query.All(&apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// --- Application routes ---

func (api *openblocksApi) applicationsHome(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	admin := api.getAdmin(c)
	authRecord := api.getAuthRecord(c)
	isAdm := admin != nil

	settings, err := api.dao.GetPblSettings().Clone()
	if err != nil {
		return errResp(c, 500, "Failed to load settings")
	}

	apps, err := api.listApps(c, false, "")
	if err != nil {
		return errResp(c, 500, "Failed to list apps")
	}

	folders := []*models.Folder{}
	api.dao.PblFolderQuery().OrderBy("updated DESC", "created DESC").All(&folders)

	// Build folder list
	folderViews := []interface{}{}
	for _, f := range folders {
		folderApps, _ := api.listApps(c, false, f.Id)
		subApps := []interface{}{}
		for _, a := range folderApps {
			subApps = append(subApps, api.createAppListItem(a, isAdm))
		}
		if !isAdm && len(subApps) == 0 {
			continue
		}
		folderViews = append(folderViews, map[string]interface{}{
			"orgId":           "ORG_ID",
			"folderId":        f.Id,
			"parentFolderId":  nil,
			"name":            f.Name,
			"createAt":        f.Created.Time().UnixMilli(),
			"subFolders":      nil,
			"subApplications": subApps,
			"createTime":      f.Created.Time().UnixMilli(),
			"lastViewTime":    f.Updated.Time().UnixMilli(),
			"visible":         true,
			"manageable":      isAdm,
			"folder":          true,
		})
	}

	appViews := []interface{}{}
	for _, a := range apps {
		appViews = append(appViews, api.createAppListItem(a, isAdm))
	}

	var userId, userName, userEmail, userUsername string
	if isAdm {
		userId = admin.Id
		userName = "Admin"
		userEmail = admin.Email
		userUsername = "ADMIN"
	} else {
		userId = authRecord.Id
		name := authRecord.GetString("name")
		if name == "NONAME" {
			name = "Unknown"
		}
		userName = name
		userEmail = authRecord.Email()
		userUsername = authRecord.Username()
	}

	themeList := []interface{}{}
	if settings.Themes != "" {
		json.Unmarshal([]byte(settings.Themes), &themeList)
	}
	libsList := []interface{}{}
	if settings.Libs != "" {
		json.Unmarshal([]byte(settings.Libs), &libsList)
	}
	pluginsList := []interface{}{}
	if settings.Plugins != "" {
		json.Unmarshal([]byte(settings.Plugins), &pluginsList)
	}

	orgCommon := map[string]interface{}{
		"themeList":          themeList,
		"defaultTheme":      settings.ThemeId,
		"preloadCSS":        settings.Css,
		"preloadJavaScript": settings.Script,
		"preloadLibs":       libsList,
		"npmPlugins":        pluginsList,
	}
	if settings.HomePageAppSlug != "" {
		orgCommon["defaultHomePage"] = settings.HomePageAppSlug
	}

	return okResp(c, map[string]interface{}{
		"user": map[string]interface{}{
			"id":        userId,
			"createdBy": "anonymousId",
			"name":      userName,
			"avatar":    nil,
			"tpAvatarLink": nil,
			"state":        "ACTIVATED",
			"isEnabled":    true,
			"isAnonymous":  false,
			"connections": []interface{}{map[string]interface{}{
				"authId": "EMAIL",
				"source": "EMAIL",
				"name":   userName,
				"avatar": nil,
				"rawUserInfo": map[string]interface{}{
					"email":    userEmail,
					"username": userUsername,
				},
				"tokens": []interface{}{},
			}},
			"hasSetNickname":         true,
			"orgTransformedUserInfo": nil,
		},
		"organization": map[string]interface{}{
			"id":                          "ORG_ID",
			"createdBy":                   "anonymousId",
			"name":                        settings.Name,
			"isAutoGeneratedOrganization": true,
			"contactName":                 nil,
			"contactEmail":                nil,
			"contactPhoneNumber":          nil,
			"source":                      nil,
			"thirdPartyCompanyId":         nil,
			"state":                       "ACTIVE",
			"commonSettings":              orgCommon,
			"logoUrl":                     settings.LogoUrl,
			"createTime":                  0,
			"authConfigs":                 api.buildAuthConfigs(),
		},
		"folderInfoViews":      folderViews,
		"homeApplicationViews": appViews,
	})
}

func (api *openblocksApi) applicationView(c echo.Context) error {
	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	info := apis.RequestInfo(c)
	if info.Admin == nil {
		if info.AuthRecord == nil {
			if !app.Public {
				return errResp(c, 401, "Unauthorized")
			}
		}
	}

	resp, err := api.createFullAppResponse(c, app)
	if err != nil {
		return errResp(c, 500, "Failed to build response")
	}
	return okResp(c, resp)
}

func (api *openblocksApi) applicationCreate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	var body struct {
		Name                  string      `json:"name"`
		EditingApplicationDSL interface{} `json:"editingApplicationDSL"`
		ApplicationType       int         `json:"applicationType"`
		FolderId              string      `json:"folderId"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	dslBytes, _ := json.Marshal(body.EditingApplicationDSL)
	dslStr := string(dslBytes)

	form := forms.NewApplicationUpsert(api.dao, &models.Application{})
	form.Name = body.Name
	form.AppDsl = dslStr
	form.EditDsl = dslStr
	form.Type = body.ApplicationType
	form.Status = "NORMAL"
	form.FolderId = body.FolderId

	app, err := form.Submit()
	if err != nil {
		return errResp(c, 400, err.Error())
	}

	resp, err := api.createFullAppResponse(c, app)
	if err != nil {
		return errResp(c, 500, "Failed to build response")
	}
	return okResp(c, resp)
}

func (api *openblocksApi) applicationUpdate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	var body struct {
		Name                  string      `json:"name"`
		EditingApplicationDSL interface{} `json:"editingApplicationDSL"`
		ApplicationType       int         `json:"applicationType"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	if body.Name != "" {
		form.Name = body.Name
	}
	if body.EditingApplicationDSL != nil {
		dslBytes, _ := json.Marshal(body.EditingApplicationDSL)
		form.EditDsl = string(dslBytes)
	}
	if body.ApplicationType > 0 {
		form.Type = body.ApplicationType
	}

	updated, err := form.Submit()
	if err != nil {
		return errResp(c, 400, err.Error())
	}

	resp, err := api.createFullAppResponse(c, updated)
	if err != nil {
		return errResp(c, 500, "Failed to build response")
	}
	return okResp(c, resp)
}

func (api *openblocksApi) applicationDelete(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	if err := api.dao.DeletePblApp(app); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

func (api *openblocksApi) applicationPublish(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	form.AppDsl = app.EditDsl

	updated, err := form.Submit()
	if err != nil {
		return errResp(c, 400, err.Error())
	}

	resp, err := api.createFullAppResponse(c, updated)
	if err != nil {
		return errResp(c, 500, "Failed to build response")
	}
	return okResp(c, resp)
}

func (api *openblocksApi) applicationsList(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	apps, err := api.listApps(c, false, "")
	if err != nil {
		return errResp(c, 500, "Failed to list apps")
	}

	isAdm := api.isAdmin(c)
	result := []interface{}{}
	for _, a := range apps {
		result = append(result, api.createAppListItem(a, isAdm))
	}
	return okResp(c, result)
}

func (api *openblocksApi) applicationsRecycleList(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	apps, err := api.listApps(c, true, "")
	if err != nil {
		return errResp(c, 500, "Failed to list apps")
	}

	result := []interface{}{}
	for _, a := range apps {
		result = append(result, api.createAppListItem(a, true))
	}
	return okResp(c, result)
}

func (api *openblocksApi) applicationRecycle(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	form.Status = "RECYCLED"
	if _, err := form.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

func (api *openblocksApi) applicationRestore(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	form.Status = "NORMAL"
	if _, err := form.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

func (api *openblocksApi) applicationPublicToAll(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	var body struct {
		PublicToAll bool `json:"publicToAll"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	form.Public = body.PublicToAll
	updated, err := form.Submit()
	if err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, updated.Public)
}

// --- Permissions ---

func (api *openblocksApi) applicationPermissionsGet(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	settings, _ := api.dao.GetPblSettings().Clone()

	permissions := []interface{}{}
	if app.AllUsers {
		permissions = append(permissions, map[string]interface{}{
			"permissionId": "all_users|GROUP",
			"type":         "GROUP",
			"id":           "all_users",
			"avatar":       "",
			"name":         "All Users",
			"role":         "viewer",
		})
	}

	for _, gId := range app.Groups {
		gName := gId
		rec, err := api.app.Dao().FindRecordById("groups", gId)
		if err == nil {
			gName = rec.GetString("name")
		}
		permissions = append(permissions, map[string]interface{}{
			"permissionId": gId + "|GROUP",
			"type":         "GROUP",
			"id":           gId,
			"name":         gName,
			"role":         "viewer",
		})
	}

	for _, uId := range app.Users {
		rec, err := api.app.Dao().FindRecordById("users", uId)
		uName := uId
		var uAvatar interface{}
		if err == nil {
			name := rec.GetString("name")
			if name != "NONAME" {
				uName = name
			}
			av := rec.GetString("avatar")
			if av != "" {
				uAvatar = "/api/files/users/" + rec.Id + "/" + av + "?thumb=100x100"
			}
		}
		permissions = append(permissions, map[string]interface{}{
			"permissionId": uId + "|USER",
			"type":         "USER",
			"id":           uId,
			"avatar":       uAvatar,
			"name":         uName,
			"role":         "viewer",
		})
	}

	groupPerms := []interface{}{}
	userPerms := []interface{}{}
	for _, p := range permissions {
		pm := p.(map[string]interface{})
		if pm["type"] == "GROUP" {
			groupPerms = append(groupPerms, p)
		} else {
			userPerms = append(userPerms, p)
		}
	}

	return okResp(c, map[string]interface{}{
		"orgName":          settings.Name,
		"groupPermissions": groupPerms,
		"userPermissions":  userPerms,
		"publicToAll":      app.Public,
		"permissions":      permissions,
	})
}

func (api *openblocksApi) applicationPermissionsUpdate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	var body struct {
		UserIds  []string `json:"userIds"`
		GroupIds []string `json:"groupIds"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	newUsers := append([]string{}, app.Users...)
	newGroups := append([]string{}, app.Groups...)

	for _, uid := range body.UserIds {
		if !slices.Contains(newUsers, uid) {
			newUsers = append(newUsers, uid)
		}
	}
	for _, gid := range body.GroupIds {
		if gid == "all_users" {
			form.AllUsers = true
			continue
		}
		if !slices.Contains(newGroups, gid) {
			newGroups = append(newGroups, gid)
		}
	}
	form.Users = newUsers
	form.Groups = newGroups

	if _, err := form.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

func (api *openblocksApi) applicationPermissionsDelete(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	slug := c.PathParam("slug")
	permId := c.PathParam("permId")

	app, err := api.dao.FindPblAppBySlug(slug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	form := forms.NewApplicationUpsert(api.dao, app)

	if permId == "all_users|GROUP" {
		form.AllUsers = false
	} else {
		parts := strings.SplitN(permId, "|", 2)
		if len(parts) == 2 {
			memberId := parts[0]
			memberType := parts[1]
			if memberType == "USER" {
				newUsers := []string{}
				for _, u := range app.Users {
					if u != memberId {
						newUsers = append(newUsers, u)
					}
				}
				form.Users = newUsers
			} else if memberType == "GROUP" {
				newGroups := []string{}
				for _, g := range app.Groups {
					if g != memberId {
						newGroups = append(newGroups, g)
					}
				}
				form.Groups = newGroups
			}
		}
	}

	if _, err := form.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

// --- Folder routes ---

func (api *openblocksApi) foldersElements(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	folderId := c.QueryParam("id")
	isAdm := api.isAdmin(c)

	result := []interface{}{}

	if folderId == "" {
		folders := []*models.Folder{}
		api.dao.PblFolderQuery().OrderBy("updated DESC", "created DESC").All(&folders)

		for _, f := range folders {
			folderApps, _ := api.listApps(c, false, f.Id)
			subApps := []interface{}{}
			for _, a := range folderApps {
				subApps = append(subApps, api.createAppListItem(a, isAdm))
			}
			if !isAdm && len(subApps) == 0 {
				continue
			}
			result = append(result, map[string]interface{}{
				"orgId":           "ORG_ID",
				"folderId":        f.Id,
				"parentFolderId":  nil,
				"name":            f.Name,
				"createAt":        f.Created.Time().UnixMilli(),
				"subFolders":      nil,
				"subApplications": subApps,
				"createTime":      f.Created.Time().UnixMilli(),
				"lastViewTime":    f.Updated.Time().UnixMilli(),
				"visible":         true,
				"manageable":      isAdm,
				"folder":          true,
			})
		}
	}

	apps, _ := api.listApps(c, false, folderId)
	for _, a := range apps {
		result = append(result, api.createAppListItem(a, isAdm))
	}

	return okResp(c, result)
}

func (api *openblocksApi) foldersCreate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	folder := &models.Folder{}
	form := forms.NewFolderUpsert(api.dao, folder)
	form.Name = body.Name

	created, err := form.Submit()
	if err != nil {
		return errResp(c, 400, err.Error())
	}

	return okResp(c, map[string]interface{}{
		"orgId":           "ORG_ID",
		"folderId":        created.Id,
		"parentFolderId":  nil,
		"name":            created.Name,
		"createAt":        created.Created.Time().UnixMilli(),
		"subFolders":      nil,
		"subApplications": []interface{}{},
		"createTime":      created.Created.Time().UnixMilli(),
		"lastViewTime":    created.Updated.Time().UnixMilli(),
		"visible":         true,
		"manageable":      true,
		"folder":          true,
	})
}

func (api *openblocksApi) foldersUpdate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	var body struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	folder, err := api.dao.FindPblFolderById(body.Id)
	if err != nil || folder == nil {
		return errResp(c, 404, "Folder not found")
	}

	form := forms.NewFolderUpsert(api.dao, folder)
	form.Name = body.Name

	updated, err := form.Submit()
	if err != nil {
		return errResp(c, 400, err.Error())
	}

	folderApps, _ := api.listApps(c, false, updated.Id)
	subApps := []interface{}{}
	for _, a := range folderApps {
		subApps = append(subApps, api.createAppListItem(a, true))
	}

	return okResp(c, map[string]interface{}{
		"orgId":           "ORG_ID",
		"folderId":        updated.Id,
		"parentFolderId":  nil,
		"name":            updated.Name,
		"createAt":        updated.Created.Time().UnixMilli(),
		"subFolders":      nil,
		"subApplications": subApps,
		"createTime":      updated.Created.Time().UnixMilli(),
		"lastViewTime":    updated.Updated.Time().UnixMilli(),
		"visible":         true,
		"manageable":      true,
		"folder":          true,
	})
}

func (api *openblocksApi) foldersMove(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	appSlug := c.PathParam("appSlug")
	targetFolderId := c.QueryParam("targetFolderId")

	app, err := api.dao.FindPblAppBySlug(appSlug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	form := forms.NewApplicationUpsert(api.dao, app)
	form.FolderId = targetFolderId
	if _, err := form.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, nil)
}

func (api *openblocksApi) foldersDelete(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	id := c.PathParam("id")
	folder, err := api.dao.FindPblFolderById(id)
	if err != nil || folder == nil {
		return errResp(c, 404, "Folder not found")
	}

	if err := api.dao.DeletePblFolder(folder); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, nil)
}

// --- Groups ---

func (api *openblocksApi) groupsList(c echo.Context) error {
	isAdm := api.isAdmin(c)
	visitorRole := "viewer"
	if isAdm {
		visitorRole = "admin"
	}

	allUsersGroup := map[string]interface{}{
		"groupId":      "all_users",
		"groupName":    "All Users",
		"allUsersGroup": true,
		"visitorRole":  visitorRole,
		"createTime":   0,
		"dynamicRule":  nil,
		"syncGroup":    false,
		"devGroup":     false,
		"syncDelete":   false,
	}

	result := []interface{}{allUsersGroup}

	groups, err := api.app.Dao().FindRecordsByFilter("groups", "", "-created", 500, 0)
	if err == nil {
		for _, g := range groups {
			result = append(result, map[string]interface{}{
				"groupId":      g.Id,
				"groupName":    g.GetString("name"),
				"allUsersGroup": false,
				"visitorRole":  visitorRole,
				"createTime":   g.Created.Time().UnixMilli(),
				"dynamicRule":  nil,
				"syncGroup":    false,
				"devGroup":     false,
				"syncDelete":   false,
			})
		}
	}

	return okResp(c, result)
}

// --- Snapshots ---

func (api *openblocksApi) snapshotView(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	id := c.PathParam("id")
	snapshot, err := api.dao.FindPblSnapshotById(id)
	if err != nil || snapshot == nil {
		return errResp(c, 404, "Snapshot not found")
	}

	var dsl interface{}
	json.Unmarshal([]byte(snapshot.Dsl), &dsl)

	return okResp(c, map[string]interface{}{
		"applicationsDsl": dsl,
		"moduleDSL":       map[string]interface{}{},
	})
}

func (api *openblocksApi) snapshotList(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	appSlug := c.PathParam("appSlug")
	app, err := api.dao.FindPblAppBySlug(appSlug, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	size, _ := strconv.Atoi(c.QueryParam("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	snapshots := []*models.Snapshot{}
	query := api.dao.PblSnapshotQuery().
		AndWhere(dbx.HashExp{"app": app.Id}).
		OrderBy("updated DESC", "created DESC").
		Limit(int64(size)).
		Offset(int64((page - 1) * size))

	query.All(&snapshots)

	var total int
	api.dao.PblSnapshotQuery().
		Select("count(*)").
		AndWhere(dbx.HashExp{"app": app.Id}).
		Row(&total)

	list := []interface{}{}
	for _, s := range snapshots {
		var ctx interface{}
		json.Unmarshal([]byte(s.Context), &ctx)
		list = append(list, map[string]interface{}{
			"snapshotId": s.Id,
			"context":    ctx,
			"createTime": s.Created.Time().UnixMilli(),
		})
	}

	return okResp(c, map[string]interface{}{
		"list":  list,
		"count": total,
	})
}

func (api *openblocksApi) snapshotCreate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	var body struct {
		ApplicationId string      `json:"applicationId"`
		Context       interface{} `json:"context"`
		Dsl           interface{} `json:"dsl"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	app, err := api.dao.FindPblAppBySlug(body.ApplicationId, nil)
	if err != nil || app == nil {
		return errResp(c, 404, "Application not found")
	}

	contextBytes, _ := json.Marshal(body.Context)
	dslBytes, _ := json.Marshal(body.Dsl)

	snapshot := &models.Snapshot{
		AppId:   app.Id,
		Context: string(contextBytes),
		Dsl:     string(dslBytes),
	}
	snapshot.MarkAsNew()
	snapshot.SetId(utils.GenerateId())
	snapshot.Created.Scan(time.Now().UTC())
	snapshot.Updated.Scan(time.Now().UTC())

	if err := api.dao.SavePblSnapshot(snapshot); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

// --- Configs ---

func (api *openblocksApi) configsView(c echo.Context) error {
	settings, err := api.dao.GetPblSettings().Clone()
	if err != nil {
		return errResp(c, 500, "Failed to load settings")
	}

	return okResp(c, map[string]interface{}{
		"authConfigs":   api.buildAuthConfigs(),
		"workspaceMode": "ENTERPRISE",
		"selfDomain":    false,
		"cookieName":    "TOKEN",
		"cloudHosting":  false,
		"featureFlag": map[string]interface{}{
			"enableCustomBranding": true,
		},
		"branding": map[string]interface{}{
			"logo":        settings.LogoUrl,
			"favicon":     settings.IconUrl,
			"brandName":   settings.Name,
			"headerColor": settings.HeaderColor,
		},
	})
}

func (api *openblocksApi) configsUpdate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	var body struct {
		Branding *struct {
			BrandName   string `json:"brandName"`
			HeaderColor string `json:"headerColor"`
			Favicon     string `json:"favicon"`
			Logo        string `json:"logo"`
		} `json:"branding"`
		Auths *models.Auths `json:"auths"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	settingsForm := forms.NewSettingsUpsert(api.dao)
	if body.Branding != nil {
		settingsForm.Name = body.Branding.BrandName
		settingsForm.HeaderColor = body.Branding.HeaderColor
		settingsForm.IconUrl = body.Branding.Favicon
		settingsForm.LogoUrl = body.Branding.Logo
	}
	if body.Auths != nil {
		settingsForm.Auths = *body.Auths
	}

	if err := settingsForm.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}

	settings, _ := api.dao.GetPblSettings().Clone()
	return okResp(c, map[string]interface{}{
		"authConfigs":   api.buildAuthConfigs(),
		"workspaceMode": "ENTERPRISE",
		"selfDomain":    false,
		"cookieName":    "TOKEN",
		"cloudHosting":  false,
		"featureFlag": map[string]interface{}{
			"enableCustomBranding": true,
		},
		"branding": map[string]interface{}{
			"logo":        settings.LogoUrl,
			"favicon":     settings.IconUrl,
			"brandName":   settings.Name,
			"headerColor": settings.HeaderColor,
		},
	})
}

// --- Organizations ---

func (api *openblocksApi) orgCommonSettings(c echo.Context) error {
	if err := api.requireAuth(c); err != nil {
		return err
	}

	settings, err := api.dao.GetPblSettings().Clone()
	if err != nil {
		return errResp(c, 500, "Failed to load settings")
	}

	themeList := []interface{}{}
	if settings.Themes != "" {
		json.Unmarshal([]byte(settings.Themes), &themeList)
	}
	libsList := []interface{}{}
	if settings.Libs != "" {
		json.Unmarshal([]byte(settings.Libs), &libsList)
	}
	pluginsList := []interface{}{}
	if settings.Plugins != "" {
		json.Unmarshal([]byte(settings.Plugins), &pluginsList)
	}

	result := map[string]interface{}{
		"themeList":          themeList,
		"defaultTheme":      settings.ThemeId,
		"preloadCSS":        settings.Css,
		"preloadJavaScript": settings.Script,
		"preloadLibs":       libsList,
		"npmPlugins":        pluginsList,
	}
	if settings.HomePageAppSlug != "" {
		result["defaultHomePage"] = settings.HomePageAppSlug
	}
	return okResp(c, result)
}

func (api *openblocksApi) orgCommonSettingsUpdate(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	renamedParams := map[string]string{
		"themeList":          "themes",
		"defaultHomePage":    "homePage",
		"defaultTheme":      "theme",
		"preloadCSS":        "css",
		"preloadJavaScript": "script",
		"preloadLibs":       "libs",
		"npmPlugins":        "plugins",
	}

	var body struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	settingsKey, ok := renamedParams[body.Key]
	if !ok {
		return errResp(c, 400, "Invalid settings key")
	}

	var valueStr string
	switch v := body.Value.(type) {
	case string:
		valueStr = v
	case nil:
		valueStr = ""
	default:
		bytes, _ := json.Marshal(v)
		valueStr = string(bytes)
	}

	settingsForm := forms.NewSettingsUpsert(api.dao)
	switch settingsKey {
	case "themes":
		settingsForm.Themes = valueStr
	case "homePage":
		settingsForm.HomePageAppSlug = valueStr
	case "theme":
		settingsForm.ThemeId = valueStr
	case "css":
		settingsForm.Css = valueStr
	case "script":
		settingsForm.Script = valueStr
	case "libs":
		settingsForm.Libs = valueStr
	case "plugins":
		settingsForm.Plugins = valueStr
	}

	if err := settingsForm.Submit(); err != nil {
		return errResp(c, 400, err.Error())
	}
	return okResp(c, true)
}

func (api *openblocksApi) orgMembers(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}

	users, err := api.app.Dao().FindRecordsByFilter("users", "", "-created", 500, 0)
	if err != nil {
		return errResp(c, 500, "Failed to list users")
	}

	members := []interface{}{}
	for _, u := range users {
		name := u.GetString("name")
		if name == "NONAME" {
			name = "Unknown"
		}
		members = append(members, map[string]interface{}{
			"userId":    u.Id,
			"name":      name,
			"avatarUrl": api.getUserAvatarUrl(u),
			"role":      "member",
			"joinTime":  u.Created.Time().UnixMilli(),
			"rawUserInfos": map[string]interface{}{
				"EMAIL": map[string]interface{}{
					"email": "Private",
				},
			},
		})
	}

	return okResp(c, map[string]interface{}{
		"visitorRole": "admin",
		"members":     members,
	})
}
