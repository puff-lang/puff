; *************** Alloc {{.N}} ***************
%pi{{.Ninstr}} = alloca i64
store i64 0, i64* %pi{{.Ninstr}}
br label %LOOP{{.Ninstr}}

LOOP{{.Ninstr}}:
    %i{{.Ninstr}} = load i64* %pi{{.Ninstr}}
    %cond{{.Ninstr}} = icmp eq i64 %i{{.Ninstr}}, {{.N}}
    br i1 %cond{{.Ninstr}}, label %DONE_LOOP{{.Ninstr}}, label %ALLOC{{.Ninstr}}

ALLOC{{.Ninstr}}:
    %node{{.Ninstr}} = call i64*(i64*)* @hAllocInd(i64* null)
    call void(i64*)* @push(i64* %node{{.Ninstr}})

    ; increment counter
    %i1{{.Ninstr}} = add i64 1, %i{{.Ninstr}}
    store i64 %i1{{.Ninstr}}, i64* %pi{{.Ninstr}}
    br label %LOOP{{.Ninstr}}

DONE_LOOP{{.Ninstr}}:

call i32 @puts(i8* getelementptr inbounds ([13 x i8]* @allocmsg, i32 0, i32 0))

