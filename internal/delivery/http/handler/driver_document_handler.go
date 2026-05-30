package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type DriverDocumentHandler struct {
	useCase usecase.DriverDocumentUseCase
}

// NewDriverDocumentHandler creates a new instance of DriverDocumentHandler.
func NewDriverDocumentHandler(useCase usecase.DriverDocumentUseCase) *DriverDocumentHandler {
	return &DriverDocumentHandler{
		useCase: useCase,
	}
}

func (h *DriverDocumentHandler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	driverID := chi.URLParam(r, "driverId")
	if driverID == "" {
		response.Error(w, http.StatusBadRequest, "driver ID is required")
		return
	}

	q := r.URL.Query()
	filter := domain.DriverDocumentFilter{
		DriverID: driverID,
		Type:     q.Get("type"),
		Status:   q.Get("status"),
	}

	docs, err := h.useCase.GetDocuments(ctx, orgID, filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve documents")
		return
	}

	response.OK(w, dto.DriverDocumentToResponseList(docs))
}

func (h *DriverDocumentHandler) GetDocumentByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "document ID is required")
		return
	}

	doc, err := h.useCase.GetDocumentByID(ctx, orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDriverDocumentNotFound) {
			response.Error(w, http.StatusNotFound, "driver document not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to retrieve document details")
		return
	}

	response.OK(w, dto.DriverDocumentToResponse(doc))
}

func (h *DriverDocumentHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	driverID := chi.URLParam(r, "driverId")
	if driverID == "" {
		response.Error(w, http.StatusBadRequest, "driver ID is required")
		return
	}

	var req dto.CreateDriverDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" || req.DocumentNumber == "" {
		response.Error(w, http.StatusBadRequest, "type and documentNumber are required")
		return
	}

	domainDoc := domain.DriverDocument{
		DriverID:         driverID,
		Type:             req.Type,
		DocumentNumber:   req.DocumentNumber,
		IssuingAuthority: req.IssuingAuthority,
		IssueDate:        req.IssueDate,
		ExpirationDate:   req.ExpirationDate,
		Notes:            req.Notes,
	}

	created, err := h.useCase.CreateDocument(ctx, orgID, domainDoc)
	if err != nil {
		if errors.Is(err, domain.ErrDriverNotFound) {
			response.Error(w, http.StatusNotFound, "driver not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create driver document")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.DriverDocumentToResponse(created), "driver document created successfully")
}

func (h *DriverDocumentHandler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "document ID is required")
		return
	}

	var req dto.UpdateDriverDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domainDoc := domain.DriverDocument{
		DocumentNumber:   req.DocumentNumber,
		IssuingAuthority: req.IssuingAuthority,
		IssueDate:        req.IssueDate,
		ExpirationDate:   req.ExpirationDate,
		Notes:            req.Notes,
	}

	updated, err := h.useCase.UpdateDocument(ctx, orgID, id, domainDoc)
	if err != nil {
		if errors.Is(err, domain.ErrDriverDocumentNotFound) {
			response.Error(w, http.StatusNotFound, "driver document not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update driver document")
		return
	}

	response.OK(w, dto.DriverDocumentToResponse(updated))
}

func (h *DriverDocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "document ID is required")
		return
	}

	err := h.useCase.DeleteDocument(ctx, orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrDriverDocumentNotFound) {
			response.Error(w, http.StatusNotFound, "driver document not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete driver document")
		return
	}

	response.OK(w, map[string]string{"message": "driver document deleted successfully"})
}

func (h *DriverDocumentHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "document ID is required")
		return
	}

	// Restrict size limit to 5MB (mitigate storage DOS vulnerability)
	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)
	err := r.ParseMultipartForm(5 * 1024 * 1024)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "multipart form parsing error or file size exceeds 5MB limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "missing file in request parameters")
		return
	}
	defer file.Close()

	updatedDoc, err := h.useCase.UploadAttachment(ctx, orgID, id, header.Filename, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		if errors.Is(err, domain.ErrDriverDocumentNotFound) {
			response.Error(w, http.StatusNotFound, "driver document not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to upload document attachment")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.DriverDocumentToResponse(updatedDoc), "file uploaded successfully")
}

func (h *DriverDocumentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	fileID := chi.URLParam(r, "fileId")
	if id == "" || fileID == "" {
		response.Error(w, http.StatusBadRequest, "document ID and file ID are required")
		return
	}

	updatedDoc, err := h.useCase.DeleteAttachment(ctx, orgID, id, fileID)
	if err != nil {
		if errors.Is(err, domain.ErrDriverDocumentNotFound) {
			response.Error(w, http.StatusNotFound, "driver document not found")
			return
		}
		if errors.Is(err, domain.ErrFileNotFound) {
			response.Error(w, http.StatusNotFound, "file attachment not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete file attachment")
		return
	}

	response.OK(w, dto.DriverDocumentToResponse(updatedDoc))
}

func (h *DriverDocumentHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	dash, err := h.useCase.GetDashboard(ctx, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load dashboard analytics")
		return
	}

	response.OK(w, dto.DocumentDashboardToResponse(dash))
}
