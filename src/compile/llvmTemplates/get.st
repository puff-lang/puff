; *************** Get ***************
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @getmsg, i32 0, i32 0))
%top.{{.Ninstr}} = call i64*()* @pop()
%tag.{{.Ninstr}} = call i64(i64*)* @getTag(i64* %top.{{.Ninstr}})

switch i64 %tag.{{.Ninstr}}, label %NOT_FOUND.{{.Ninstr}} [ i64 1, label %NUM.{{.Ninstr}} ]

NUM.{{.Ninstr}}:
    %pnum.{{.Ninstr}} = call i64*(i64*)* @getNumPtr(i64* %top.{{.Ninstr}})
    %num.{{.Ninstr}} = load i64* %pnum.{{.Ninstr}}

    call void(i64)* @pushV(i64 %num.{{.Ninstr}})
    br label %DONE_GET.{{.Ninstr}}

NOT_FOUND.{{.Ninstr}}:
    %ps.{{.Ninstr}} = getelementptr [63 x i8]* @getErrStr, i64 0, i64 0
    call i64 (i8 *, ...)* @printf(i8* %ps.{{.Ninstr}})
    br label %DONE_GET.{{.Ninstr}}

DONE_GET.{{.Ninstr}}:


