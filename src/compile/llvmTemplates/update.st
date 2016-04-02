; *************** Update {{.N}} ***************
%top{{.Ninstr}} = call i64*()* @pop()

; update the nth node on the stack to hold the same value as the top node
%vsp{{.Ninstr}} = load i64* @sp
%n1{{.Ninstr}} = add i64 {{.N}}, 1
%rootIndex{{.Ninstr}} = sub i64 %vsp{{.Ninstr}}, %n1{{.Ninstr}}
%toUpdate{{.Ninstr}} = call i64**(i64)* @getItemPtr(i64 %rootIndex{{.Ninstr}})

; create ind node on the heap
%ind{{.Ninstr}} = call i64*(i64*)* @hAllocInd(i64* %top{{.Ninstr}})

store i64* %ind{{.Ninstr}}, i64** %toUpdate{{.Ninstr}}
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @updatemsg, i32 0, i32 0))