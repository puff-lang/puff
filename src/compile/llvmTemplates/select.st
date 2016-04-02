; *************** Select {{.N}} $k$ ***************
%pconstr.{{.Ninstr}} = call i64*()* @pop()

; we assume that element on the top of the stack is a constructor
%pargs.{{.Ninstr}} = call i64**(i64*)* @getConstrArgsPtr(i64* %pconstr.{{.Ninstr}})

%parg.{{.Ninstr}} = call i64**(i64, i64**)* @incPtr(i64 $k$, i64** %pargs.{{.Ninstr}})
%arg.{{.Ninstr}} = load i64** %parg.{{.Ninstr}}

call void(i64*)* @push(i64* %arg.{{.Ninstr}})

