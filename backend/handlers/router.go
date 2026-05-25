package handlers

import (
	"database/sql"
	"net/http"

	"github.com/brian/config-generation/backend/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(db *sql.DB, jwtSecret []byte) chi.Router {
	return NewRouterWithAuthConfig(db, DefaultAuthConfig(jwtSecret))
}

func NewRouterWithAuthConfig(db *sql.DB, authConfig AuthConfig) chi.Router {
	r := chi.NewRouter()

	// Prometheus metrics middleware — wraps all routes.
	r.Use(middleware.PrometheusMetrics)

	// ── Public routes (no JWT) ─────────────────────────────────────────────

	// Liveness probe: always 200 if the process is running.
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Readiness probe: 200 only when the DB is reachable.
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Prometheus scrape endpoint.
	r.Handle("/metrics", promhttp.Handler())

	auth := &AuthHandler{DB: db, Config: authConfig}
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/config", auth.ConfigResponse)
		r.Get("/me", auth.Me)
		r.Post("/session", auth.Session)
		r.Post("/logout", auth.Logout)
		r.Post("/register", auth.Register)
		r.Post("/login", auth.Login)
		r.Get("/oidc/login", auth.OIDCLogin)
		r.Get("/oidc/callback", auth.OIDCCallback)
	})

	proj := &ProjectHandler{DB: db}
	env := &EnvironmentHandler{DB: db}
	tmpl := &TemplateHandler{DB: db}
	vals := &ValuesHandler{DB: db}
	gv := &GlobalValuesHandler{DB: db}
	role := &RoleHandler{DB: db}
	pr := &PullRequestHandler{DB: db}
	deploy := &DeploymentHandler{DB: db}

	perm := middleware.RequirePermission
	param := middleware.URLParam

	// Protected routes: JWT required.
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authConfig.JWTSecret, authConfig.SessionCookieName))
		r.Use(middleware.CSRFProtection(csrfCookieName))

		r.Route("/api", func(r chi.Router) {

			// --- Projects ---
			r.Route("/projects", func(r chi.Router) {
				r.With(perm(db, "create", "project", nil, nil, nil)).
					Post("/", proj.Create)
				r.Get("/", proj.List)

				r.Route("/{projectName}", func(r chi.Router) {
					r.Get("/", proj.Get)
					r.With(perm(db, "delete", "project", param("projectName"), nil, nil)).
						Delete("/", proj.Delete)

					// --- Templates ---
					r.Route("/templates", func(r chi.Router) {
						r.With(perm(db, "read", "project_templates", param("projectName"), nil, nil)).
							Get("/", tmpl.ListForProject)
						r.With(perm(db, "write", "project_templates", param("projectName"), nil, nil)).
							Post("/", tmpl.Create)

						r.Route("/{templateName}", func(r chi.Router) {
							r.With(perm(db, "read", "project_templates", param("projectName"), nil, nil)).
								Get("/", tmpl.GetLatest)
							r.With(perm(db, "read", "project_templates", param("projectName"), nil, nil)).
								Get("/variables", tmpl.Variables)

							r.Route("/versions", func(r chi.Router) {
								r.With(perm(db, "read", "project_templates", param("projectName"), nil, nil)).
									Get("/", tmpl.ListVersions)
								r.With(perm(db, "write", "project_templates", param("projectName"), nil, nil)).
									Post("/", tmpl.AppendVersion)
								r.With(perm(db, "read", "project_templates", param("projectName"), nil, nil)).
									Get("/{versionID}", tmpl.GetVersion)
							})
						})
					})

					// --- Project-level variables (union of all templates) ---
					r.With(perm(db, "read", "project_templates", param("projectName"), nil, nil)).
						Get("/variables", tmpl.ProjectVariables)

					// --- Values per (project, env) ---
					r.With(perm(db, "create", "env_values", param("projectName"), nil, nil)).
						Post("/values", vals.Create)

					r.Route("/envs/{envName}", func(r chi.Router) {
						r.Route("/values", func(r chi.Router) {
							r.With(perm(db, "read", "project_values", param("projectName"), param("envName"), nil)).
								Get("/", vals.GetLatest)

							r.Route("/versions", func(r chi.Router) {
								r.With(perm(db, "write", "project_values", param("projectName"), param("envName"), nil)).
									Post("/", vals.AppendVersion)
								r.With(perm(db, "read", "project_values", param("projectName"), param("envName"), nil)).
									Get("/{versionID}", vals.GetVersion)
							})
						})

						// --- Deployments ---
						r.Route("/deploy", func(r chi.Router) {
							r.Post("/preview", deploy.Preview)
							r.Post("/", deploy.Execute)
						})
						r.Get("/deployments/latest", deploy.GetLatest)
					})

					// --- Environments (project-scoped) ---
					r.Route("/environments", func(r chi.Router) {
						r.Get("/", env.List)
						r.Get("/{envName}", env.Get)
					})

					// --- Roles (project-scoped) ---
					r.Route("/roles", func(r chi.Router) {
						r.With(perm(db, "grant", "", param("projectName"), nil, nil)).
							Post("/", role.Create)
						r.With(perm(db, "grant", "", param("projectName"), nil, nil)).
							Get("/", role.ListForProject)
					})
				})
			})

			// --- Global Values ---
			r.Route("/global-values", func(r chi.Router) {
				r.Get("/", gv.List)
				r.Post("/", gv.Create)

				r.Route("/{name}", func(r chi.Router) {
					r.With(perm(db, "read", "global_values", nil, nil, param("name"))).
						Get("/", gv.GetLatest)

					r.Route("/versions", func(r chi.Router) {
						r.With(perm(db, "read", "global_values", nil, nil, param("name"))).
							Get("/", gv.ListVersions)
						r.With(perm(db, "write", "global_values", nil, nil, param("name"))).
							Post("/", gv.AppendVersion)
						r.With(perm(db, "read", "global_values", nil, nil, param("name"))).
							Get("/{versionID}", gv.GetVersion)
					})

					// --- Roles (global values scoped) ---
					r.Route("/roles", func(r chi.Router) {
						r.With(perm(db, "grant", "global_values", nil, nil, param("name"))).
							Get("/", role.ListForGlobalValues)
					})
				})
			})

			// --- Pull Requests ---
			r.Route("/pull-requests", func(r chi.Router) {
				r.Post("/", pr.Create)
				r.Get("/", pr.List)
				r.Get("/{prID}", pr.Get)
				r.Post("/{prID}/close", pr.Close)
				r.Post("/{prID}/merge", pr.Merge)
				r.Post("/{prID}/approve", pr.Approve)
				r.Post("/{prID}/withdraw-approval", pr.WithdrawApproval)
				r.Post("/{prID}/submit", pr.SubmitDraft)
			})

			// --- Workspace ---
			r.Route("/workspace/{projectName}", func(r chi.Router) {
				r.Get("/draft", pr.GetActiveDraft)
				r.Post("/stage", pr.StageChange)
			})

			// --- Roles (non-project-scoped operations by role ID) ---
			// Permission checks happen inside handlers since we need to look up the role's project first.
			r.Route("/roles/{roleID}", func(r chi.Router) {
				r.Put("/permissions", role.EditPermissions)
				r.Delete("/", role.Delete)
				r.Post("/members", role.AssignUser)
				r.Delete("/members/{userID}", role.RemoveUser)
			})

		})
	})

	return r
}
