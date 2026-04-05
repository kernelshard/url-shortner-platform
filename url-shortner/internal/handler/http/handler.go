package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/kernelshard/url-shortner-platform/internal/repository"
	"github.com/kernelshard/url-shortner-platform/internal/service"
)

type createRequest struct {
	URL string `json:"url"`
}

type createResponse struct {
	ShortURL string `json:"short_url"`
}

type Handler struct {
	svc service.LinkService
}

// NewHandler creates a new HTTP handler with the given link service.
func NewHandler(svc service.LinkService) *Handler {
	return &Handler{svc: svc}
}

// CreateShortURL handles the creation of a short URL from an original URL.
func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	// decode and validate the request body, return 400 if invalid
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// validate the URL format, return 400 if invalid
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	// call the service to create a short URL,
	// handle errors and return appropriate responses
	link, err := h.svc.Create(r.Context(), req.URL)
	if err != nil {
		// log.Printf("failed to create short URL: %v", err)
		// http.Error(w, "failed to create short URL", http.StatusInternalServerError)
		// return
		switch err {
		case repository.ErrLinkAlreadyExists:
			// idempotent case - return success
			// call GetByURL to return existing
			resp := createResponse{
				ShortURL: "http://localhost:8080/" + link.ShortCode,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(resp)
			if err != nil {
				// log later
				return
			}
		case repository.ErrShortCodeConflict:
			http.Error(w, "conflict, retry", http.StatusConflict)
			return

		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	resp := createResponse{
		ShortURL: "http://localhost:8080/" + link.ShortCode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// log later
		return
	}
}

// Redirect handles the redirection from a short URL to the original URL.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("code")

	link, err := h.svc.GetByShortCode(r.Context(), shortCode)
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		log.Printf("error fetching code=%s: %v", shortCode, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, link.OriginalURL, http.StatusFound)
}

// GetByShortCode handles fetching a link by its short code.
// It returns the original URL and short code if found, or a 404 error if not found.
func (h *Handler) GetByShortCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	link, err := h.svc.GetByShortCode(r.Context(), code)
	// Handle errors: link not found or other repository errors.
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		log.Printf("error fetching code=%s: %v", code, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// successful response
	resp := struct {
		OriginalURL string `json:"original_url"`
		ShortCode   string `json:"short_code"`
	}{
		OriginalURL: link.OriginalURL,
		ShortCode:   link.ShortCode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// log later
		return
	}
}
