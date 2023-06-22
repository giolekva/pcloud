package welcome

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/giolekva/pcloud/core/installer"
)

//go:embed index.html
var indexHtml string

//go:embed static/*
var staticAssets embed.FS

type Server struct {
	port int
	repo installer.RepoIO
}

func NewServer(port int, repo installer.RepoIO) *Server {
	return &Server{
		port,
		repo,
	}
}

func (s *Server) Start() {
	e := echo.New()
	e.StaticFS("/static", echo.MustSubFS(staticAssets, "static"))
	e.POST("/create-admin-account", s.createAdminAccount)
	e.GET("/", s.createAdminAccountForm)
	log.Fatal(e.Start(fmt.Sprintf(":%d", s.port)))
}

func (s *Server) createAdminAccountForm(c echo.Context) error {
	return c.HTML(http.StatusOK, indexHtml)
}

type createAdminAccountReq struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"` // TODO(giolekva): actually use this
	GandiAPIToken string `json:"gandiAPIToken,omitempty"`
	SecretToken   string `json:"secretToken,omitempty"`
}

func (s *Server) createAdminAccount(c echo.Context) error {
	var req createAdminAccountReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return err
	}
	// TODO(giolekva): accounts-ui create user req
	{
		appManager, err := installer.NewAppManager(s.repo)
		if err != nil {
			return err
		}
		appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
		{
			app, err := appsRepo.Find("certificate-issuer-private")
			if err != nil {
				return err
			}
			if err := appManager.Install(*app, map[string]any{
				"GandiAPIToken": req.GandiAPIToken,
			}); err != nil {
				return err
			}
		}
		{
			app, err := appsRepo.Find("tailscale-proxy")
			if err != nil {
				return err
			}
			if err := appManager.Install(*app, map[string]any{
				"Username": req.Username,
				"IPSubnet": "10.1.0.0/24", // TODO(giolekva): this should be taken from the config generated during new env creation
			}); err != nil {
				return err
			}
			// TODO(giolekva): headscale accept routes
		}
	}
	return c.String(http.StatusOK, "OK")
}
