; *************** {{.Instr}} ***************
%a.{{.Ninstr}} = call i64()* @popV()
%b.{{.Ninstr}} = call i64()* @popV()
%res.{{.Ninstr}} = {{.Instr}} i64 %a.{{.Ninstr}}, %b.{{.Ninstr}}
call void(i64)* @pushV(i64 %res.{{.Ninstr}})

