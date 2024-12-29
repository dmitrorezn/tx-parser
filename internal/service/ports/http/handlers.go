package httpport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dmitrorezn/tx-parser/internal/domain"
	"github.com/dmitrorezn/tx-parser/internal/service"
)

type Handler struct {
	service service.Servicer
	http.Handler
}

const (
	addressParam = "address"
)

func NewHandler(svc service.Servicer) *Handler {
	h := &Handler{
		service: svc,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /getCurrentBlock", h.GetCurrentBlock)
	mux.HandleFunc("POST /subscribe", h.Subscribe)
	mux.HandleFunc(fmt.Sprintf("GET /getTransactions/{%s}", addressParam), h.GetTransactions)

	h.Handler = mux

	return h
}

type errorStatus = struct {
	err        error
	statusCode int
	msg        string
}

var errorsList = []errorStatus{
	{
		err:        domain.ErrAddressNotSubscribed,
		statusCode: http.StatusNotFound,
		msg:        "not found subscriber",
	},
	{
		err:        domain.ErrNoTransactions,
		statusCode: http.StatusNotFound,
		msg:        "not found transactions",
	},
	{
		err:        domain.ErrAddressAlreadySubscribed,
		statusCode: http.StatusConflict,
		msg:        "address already subscribed",
	},
	{
		err:        domain.ErrInvalidAddress,
		statusCode: http.StatusBadRequest,
		msg:        "invalid address",
	},
}

type ErrorResponse struct {
	Err string `json:"error"`
	Msg string `json:"msg"`
}

func handleError(w http.ResponseWriter, err error) {

	var errStatus = errorStatus{
		statusCode: http.StatusBadRequest,
		msg:        "UNKNOWN ERROR",
	}
	for _, e := range errorsList {
		if errors.Is(err, e.err) {
			errStatus = e

			break
		}
	}
	w.WriteHeader(errStatus.statusCode)
	err = json.NewEncoder(w).Encode(ErrorResponse{
		Msg: errStatus.msg,
		Err: err.Error(),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

type CurrentBlock struct {
	CurrentBlockHeight int `json:"currentBlockHeight"`
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func (h *Handler) GetCurrentBlock(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, CurrentBlock{
		CurrentBlockHeight: h.service.GetCurrentBlock(),
	})
}

type SubscribeRequest struct {
	Address string `json:"address"`
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var request SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		handleError(w, err)

		return
	}
	addr := domain.Address(request.Address)
	if err := h.service.Subscribe(r.Context(), addr); err != nil {
		handleError(w, err)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	addr := domain.Address(r.PathValue(addressParam))
	txs, err := h.service.GetTransactions(r.Context(), addr)
	if err != nil {
		handleError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, txs)
}
