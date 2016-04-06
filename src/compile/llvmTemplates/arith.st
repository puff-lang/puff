; *************** {{.Instr}} ***************
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @arithmsg, i32 0, i32 0))
%a.{{.Ninstr}} = call i64()* @popV()
%b.{{.Ninstr}} = call i64()* @popV()
%res.{{.Ninstr}} = {{.Instr}} i64 %a.{{.Ninstr}}, %b.{{.Ninstr}}
call void(i64)* @pushV(i64 %res.{{.Ninstr}})

