// web/backend/api/vault.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sipeed/picoclaw/pkg/vault"
)

func RegisterVaultRoutes(r *gin.Engine, workspaceDir string) {
	vaultPath := workspaceDir + "/vault"
	store := vault.NewVaultStore(vaultPath)

	r.GET("/api/vault/list", func(c *gin.Context) {
		// TODO: Implement list logic
		c.JSON(http.StatusOK, gin.H{"notes": []string{}})
	})

	r.GET("/api/vault/note", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
			return
		}
		fm, body, err := store.ReadNote(path)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"frontmatter": fm, "content": body})
	})

	r.GET("/api/vault/tags", func(c *gin.Context) {
		// TODO: Implement tag listing
		c.JSON(http.StatusOK, gin.H{"tags": []string{}})
	})

	r.GET("/api/sessions/search", func(c *gin.Context) {
		// TODO: Implement session search
		c.JSON(http.StatusOK, gin.H{"sessions": []string{}})
	})

	r.GET("/api/tools/skills", func(c *gin.Context) {
		// TODO: Implement tool skills listing
		c.JSON(http.StatusOK, gin.H{"skills": []string{}})
	})
}
