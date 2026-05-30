package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nova/ast"
	"strings"
)

// ExecuteServer starts a blocking HTTP server for the given ServerStatement.
// It is called from the interpreter's execStmt.
func ExecuteServer(s *ast.ServerStatement, env *Environment, interp *Interpreter) error {
	mux := http.NewServeMux()

	for _, route := range s.Routes {
		r := route // capture for closure
		mux.HandleFunc(r.Path, func(w http.ResponseWriter, req *http.Request) {
			if !strings.EqualFold(req.Method, r.Method) && r.Method != "any" {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			// Build request object available inside the handler as `request`
			reqMap := map[string]*Value{
				"method": StringVal(req.Method),
				"path":   StringVal(req.URL.Path),
				"query":  buildQueryMap(req),
			}
			if req.Body != nil {
				defer req.Body.Close()
				body, _ := io.ReadAll(req.Body)
				reqMap["body"] = StringVal(string(body))
				// Attempt JSON parse of body → request.json
				var parsed any
				if json.Unmarshal(body, &parsed) == nil {
					reqMap["json"] = anyToValue(parsed)
				}
			}
			reqVal := &Value{
				Type:     TypeMap,
				MapVal:   reqMap,
				MapOrder: []string{"method", "path", "query", "body"},
			}

			handlerEnv := NewEnclosed(env)
			handlerEnv.Set("request", reqVal)

			result, err := interp.execStmts(r.Body, handlerEnv)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Unwrap return signal
			var responseVal *Value
			if isControl(result, "__return__") && len(result.ListVal) > 0 {
				responseVal = result.ListVal[0]
			} else if result != nil {
				responseVal = result
			} else {
				responseVal = Null
			}

			writeResponse(w, responseVal)
		})
	}

	addr := fmt.Sprintf(":%d", s.Port)
	fmt.Printf("Nova server listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func buildQueryMap(req *http.Request) *Value {
	m := &Value{Type: TypeMap, MapVal: make(map[string]*Value)}
	for k, vs := range req.URL.Query() {
		m.MapVal[k] = StringVal(strings.Join(vs, ","))
		m.MapOrder = append(m.MapOrder, k)
	}
	return m
}

// WriteHTTPResponse is the public entry-point used by the VM's server handler.
func WriteHTTPResponse(w http.ResponseWriter, v *Value) { writeResponse(w, v) }

func writeResponse(w http.ResponseWriter, v *Value) {
	switch v.Type {
	case TypeMap, TypeList:
		w.Header().Set("Content-Type", "application/json")
		data, err := valueToJSON(v) // defined in modules.go
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(data) //nolint:errcheck
	case TypeNull:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, v.String())
	}
}
