package api

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/NicolauFlamel/portifolio-project-backend/internal/config"
	"github.com/NicolauFlamel/portifolio-project-backend/internal/services"
)

type Handler struct {
	docSvc *services.DocService
}

func NewHandler() *Handler {
	cfg := config.New()
	return &Handler{
		docSvc: services.NewDocService(cfg),
	}
}

// @Summary Health Check
// @Description Returns OK if API is running
// @Tags health
// @Success 200 {object} map[string]bool
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	c.JSON(200, gin.H{"ok": true})
}

// @Summary List blockchain documents
// @Description Returns all documents stored in the chosen channel/org
// @Tags documents
// @Produce json
// @Param channel query string false "Fabric Channel (default: union-channel)"
// @Param org     query string false "Organization MSPID (default: org1)"
// @Success 200 {array} object
// @Router /api/docs [get]
func (h *Handler) GetAllDocs(c *gin.Context) {
	var q ListDocumentsQuery
	_ = c.BindQuery(&q)

	out, err := h.docSvc.GetAll(q.Channel, q.Org)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Data(200, "application/json", out)
}

// @Summary Get document by ID
// @Description Retrieves one document from the blockchain
// @Tags documents
// @Produce json
// @Param id      path string true "Document ID"
// @Param channel query string false "Fabric Channel"
// @Param org     query string false "Organization"
// @Success 200 {object} map[string]interface{}
// @Router /api/docs/{id} [get]
func (h *Handler) GetDoc(c *gin.Context) {
	var q GetDocumentQuery
	_ = c.BindQuery(&q)
	id := c.Param("id")

	out, err := h.docSvc.Get(q.Channel, q.Org, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(404, gin.H{"error": "document not found"})
		} else {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	c.Data(200, "application/json", out)
}

// @Summary Create a document
// @Description Writes a new document to Fabric (endorsed + committed)
// @Tags documents
// @Accept json
// @Produce json
// @Param channel query string false "Fabric Channel"
// @Param org     query string false "Organization"
// @Param document body CreateDocumentDTO true "Document Payload"
// @Success 201 {object} map[string]string
// @Router /api/docs [post]
func (h *Handler) CreateDoc(c *gin.Context) {
	channel := c.Query("channel")
	org := c.Query("org")

	var dto CreateDocumentDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(400, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	body, _ := json.Marshal(dto)

	txID, err := h.docSvc.Create(channel, org, body)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"txID": txID})
}
