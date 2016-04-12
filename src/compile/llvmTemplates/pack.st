; *************** Pack ***************
%ptop.{{.Ninstr}} = call i64** @getTopPtr()
%pconstr.{{.Ninstr}} = call i64*(i64, i64, i64**)* @hAllocConstr(i64 {{.Tag}}, i64 {{.Arity}}, i64** %ptop.{{.Ninstr}})
call void(i64)* @popn(i64 {{.Arity}})
call void(i64*)* @push(i64* %pconstr.{{.Ninstr}})

