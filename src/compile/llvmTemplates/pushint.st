; *************** Pushint {{.N}} ***************
; create the num node on the heap
%ptag{{.Ninstr}} = call i64*(i64)* @hAllocNum(i64 {{.N}})

; push node address onto the stack
call void(i64*)* @push(i64* %ptag{{.Ninstr}})
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @pushintmsg, i32 0, i32 0))

