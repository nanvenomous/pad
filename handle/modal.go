package handle

import (
	"net/http"

	"github.com/nanvenomous/pad/ui"
)

func init() {
	setupFuncs = append(setupFuncs, func(mux *http.ServeMux) {

		mux.HandleFunc("/modal", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				stts, err := render(w, r,
					ui.EmptyModalPopover(ui.PropsEmptyModalPopover{}),
				)
				if err != nil {
					errorHTTP(w, stts, err)
				}
				return
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
		})

	})
}
