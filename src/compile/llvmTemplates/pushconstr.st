; *************** Pushconstr {{.Tag}} {{.Arity}}***************
; TODO: update map so that the same constructor is not allocated twice

%pglobal{{.Ninstr}} = call i64*(i64, void()*)* @hAllocGlobal(i64 {{.Arity}}, void()* @_constr_{{.Tag}}_{{.Arity}})
call void(i64*)* @push(i64* %pglobal{{.Ninstr}})

