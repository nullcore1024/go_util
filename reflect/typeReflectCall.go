package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "gimserver/logger"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// Precompute the reflect type for context.
var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	// numCalls   uint
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

type RpcServiceImpl struct {
	serviceMap map[string]*service
}

func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			fmt.Println("method not export")
			continue
		}
		// Method needs four ins: receiver, context.Context, *args, *reply.
		if mtype.NumIn() != 4 {
			if reportErr {
				log.Info("method", mname, " has wrong number of ins:", mtype.NumIn())
			}
			continue
		}
		// First arg must be context.Context
		ctxType := mtype.In(1)
		if !ctxType.Implements(typeOfContext) {
			if reportErr {
				log.Info("method", mname, " must use context.Context as the first parameter")
			}
			continue
		}

		// Second arg need not be a pointer.
		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Info(mname, " parameter type not exported: ", argType)
			}
			continue
		}
		// Third arg must be a pointer.
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Info("method", mname, " reply type not a pointer:", replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Info("method", mname, " reply type not exported:", replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Info("method", mname, " has wrong number of outs:", mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Info("method", mname, " returns ", returnType.String(), " not error")
			}
			continue
		}
		fmt.Println("register method:", mname)
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}

		argsReplyPools.Init(argType)
		argsReplyPools.Init(replyType)
	}
	return methods
}

func (s *service) call(ctx context.Context, mtype *methodType, argv, replyv reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			//log.Errorf("failed to invoke service: %v, stacks: %s", r, string(debug.Stack()))
			err = fmt.Errorf("[service internal error]: %v, method: %s, argv: %+v",
				r, mtype.method.Name, argv.Interface())
			log.Err(err)
		}
	}()

	fmt.Println("1")
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, reflect.ValueOf(ctx), argv, replyv})
	fmt.Println("2")
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	if errInter != nil {
		fmt.Println("3")
		return errInter.(error)
	}

	return nil
}

func (s *RpcServiceImpl) handleRequest(ctx context.Context, body []byte, serviceName, methodName string) (res []byte, err error) {
	service := s.serviceMap[serviceName]
	if service == nil {
		err = errors.New("rpcx: can't find service " + serviceName)
		return nil, err
	}
	mtype := service.method[methodName]
	if mtype == nil {
		err = errors.New("rpcx: can't find method " + methodName)
		return nil, err
	}

	var argv = argsReplyPools.Get(mtype.ArgType)
	replyv := argsReplyPools.Get(mtype.ReplyType)
	/*
		argv := reflect.New(mtype.ArgType).Elem().Interface()
		replyv := reflect.New(mtype.ReplyType).Elem().Interface()
	*/
	if err := json.Unmarshal(body, &argv); err != nil {
		return nil, err
	}

	if mtype.ArgType.Kind() != reflect.Ptr {
		fmt.Println("not ptr")
		err = service.call(ctx, mtype, reflect.ValueOf(argv).Elem(), reflect.ValueOf(replyv))
	} else {
		fmt.Println("is ptr")
		err = service.call(ctx, mtype, reflect.ValueOf(argv), reflect.ValueOf(replyv))
	}
	return json.Marshal(replyv)
}

func (s *RpcServiceImpl) register(rcvr interface{}, name string, useName bool) (string, error) {
	if s.serviceMap == nil {
		s.serviceMap = make(map[string]*service)
	}

	service := new(service)
	service.typ = reflect.TypeOf(rcvr)
	service.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(service.rcvr).Type().Name() // Type
	if useName {
		sname = name
	}
	if sname == "" {
		errorStr := "rpcx.Register: no service name for type " + service.typ.String()
		log.Err(errorStr)
		return sname, errors.New(errorStr)
	}
	if !useName && !isExported(sname) {
		errorStr := "rpcx.Register: type " + sname + " is not exported"
		log.Err(errorStr)
		return sname, errors.New(errorStr)
	}
	service.name = sname

	// Install the methods
	service.method = suitableMethods(service.typ, true)

	if len(service.method) == 0 {
		var errorStr string

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(service.typ), false)
		if len(method) != 0 {
			errorStr = "rpcx.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			errorStr = "rpcx.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Err(errorStr)
		return sname, errors.New(errorStr)
	}
	fmt.Println("register service:", service.name)
	s.serviceMap[service.name] = service
	return sname, nil
}

type HelloImp struct {
}

type RpcReq struct {
	SeqId uint32 `json:seqId`
	Num   uint32 `json:num`
}

type RpcRes struct {
	SeqId uint32 `json:seqId`
	Num   uint32 `json:num`
}

func (thiz *HelloImp) OnRpcCall(ctx context.Context, req *RpcReq, res *RpcRes) error {
	fmt.Println("seqid:", req.SeqId, "num:", req.Num)
	res.SeqId = req.SeqId
	res.Num = req.Num + 1
	return nil
}

func main() {
	smap := RpcServiceImpl{}
	hello := &HelloImp{}
	if name, err := smap.register(hello, "", false); err != nil {
		fmt.Println("register fail:", name, err)
		return
	}
	proto := RpcReq{
		SeqId: 12121,
		Num:   2222,
	}
	req, err := json.Marshal(&proto)
	if err != nil {
		fmt.Println("req json err:", err)
		return
	}

	res, err := smap.handleRequest(context.Background(), req, "HelloImp", "OnRpcCall")
	if err != nil {
		fmt.Println("err:", err)
	} else {
		fmt.Println("res", string(res))
	}
}

var UsePool bool

// Reset defines Reset method for pooled object.
type Reset interface {
	Reset()
}

var argsReplyPools = &typePools{
	pools: make(map[reflect.Type]*sync.Pool),
	New: func(t reflect.Type) interface{} {
		var argv reflect.Value

		if t.Kind() == reflect.Ptr { // reply must be ptr
			argv = reflect.New(t.Elem())
		} else {
			argv = reflect.New(t)
		}

		return argv.Interface()
	},
}

type typePools struct {
	mu    sync.RWMutex
	pools map[reflect.Type]*sync.Pool
	New   func(t reflect.Type) interface{}
}

func (p *typePools) Init(t reflect.Type) {
	tp := &sync.Pool{}
	tp.New = func() interface{} {
		return p.New(t)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pools[t] = tp
}

func (p *typePools) Put(t reflect.Type, x interface{}) {
	if !UsePool {
		return
	}
	if o, ok := x.(Reset); ok {
		o.Reset()
	}

	p.mu.RLock()
	pool := p.pools[t]
	p.mu.RUnlock()
	pool.Put(x)
}

func (p *typePools) Get(t reflect.Type) interface{} {
	if !UsePool {
		return p.New(t)
	}
	p.mu.RLock()
	pool := p.pools[t]
	p.mu.RUnlock()

	return pool.Get()
}
