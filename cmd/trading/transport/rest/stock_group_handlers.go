package rest

import (
	"encoding/json"
	"net/http"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/service"
	"strings"

	"github.com/gorilla/mux"
)

type StockGroupHandler struct {
	Service              *service.StockGroupService
	StockUniverseService *service.StockUniverseService
}

func NewStockGroupHandler(
	svc *service.StockGroupService,
	stockUniverseService *service.StockUniverseService,
) *StockGroupHandler {
	return &StockGroupHandler{
		Service:              svc,
		StockUniverseService: stockUniverseService,
	}
}

type createGroupRequest struct {
	EntryType string   `json:"entryType"`
	StockIDs  []string `json:"stockIds"`
}

type editGroupRequest struct {
	StockIDs []string `json:"stockIds"`
}

func (h *StockGroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	group, err := h.Service.CreateGroup(r.Context(), req.EntryType, req.StockIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(group)
}

func (h *StockGroupHandler) EditGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req editGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Service.EditGroup(r.Context(), id, req.StockIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockGroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.Service.DeleteGroup(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockGroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	entryType := r.URL.Query().Get("entryType")
	status := domain.StockGroupStatus(strings.ToUpper(r.URL.Query().Get("status")))
	groups, err := h.Service.ListGroupsEnriched(
		r.Context(),
		entryType,
		status,
		h.StockUniverseService,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(groups)
}

func (h *StockGroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	group, err := h.Service.ListGroupsEnriched(
		r.Context(),
		"",
		domain.GroupPending,
		h.StockUniverseService,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, g := range group {
		if g.ID == id {
			json.NewEncoder(w).Encode(g)
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (h *StockGroupHandler) ExecuteGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	group, err := h.Service.ListGroupsEnriched(
		r.Context(),
		"",
		domain.GroupPending,
		h.StockUniverseService,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, stock := range group.Stocks {
		// here stock id, symbol, instryment_key and exchange token are present.
		//
	}
	w.WriteHeader(http.StatusNotFound)
}
