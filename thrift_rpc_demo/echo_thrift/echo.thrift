namespace go echo

struct EchoReq {
    1: string msg;
}

struct EchoReq2 {
    1: string msg;
}

struct EchoRes {
    1: string msg;
}

service Echo {
    EchoRes sayEcho(1: EchoReq req);
}
