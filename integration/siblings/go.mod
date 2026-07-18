module github.com/faustbrian/go-http-middleware/integration/siblings

go 1.26.5

require (
	github.com/faustbrian/go-http-middleware v0.0.0
	github.com/faustbrian/go-router v0.0.0-20260718023540-3aa46322f030
	github.com/faustbrian/go-service v0.0.0-20260716065542-258473c23446
)

require github.com/felixge/httpsnoop v1.1.0 // indirect

replace github.com/faustbrian/go-http-middleware => ../..
