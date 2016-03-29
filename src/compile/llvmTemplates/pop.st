; *************** Pop {{.N}} ***************
%vsp{{.Ninstr}} = load i64* @sp

; update the stack pointer
%vsp1{{.Ninstr}} = sub i64 %vsp{{.Ninstr}}, {{.N}}
store i64 %vsp1{{.Ninstr}}, i64* @sp
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @popmsg, i32 0, i32 0))