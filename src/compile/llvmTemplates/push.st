; *************** Push {{.Ninstr}}***************
%vsp{{.Ninstr}} = load i64* @sp
%tmp{{.Ninstr}} = add i64 {{.N}}, 1
%n1{{.Ninstr}} = sub i64 %vsp{{.Ninstr}}, %tmp{{.Ninstr}}
%paddr{{.Ninstr}} = getelementptr [1000 x i64*]* @stack, i64 0, i64 %n1{{.Ninstr}}
%addr{{.Ninstr}} = load i64** %paddr{{.Ninstr}}

call void(i64*)* @push(i64* %addr{{.Ninstr}})


call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @pushmsg, i32 0, i32 0))