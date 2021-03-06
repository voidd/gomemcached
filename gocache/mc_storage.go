package main

import (
	"encoding/binary"
	"github.com/voidd/gomemcached"
	"log"
)

type storage struct {
	data map[string]gomemcached.MCItem
	cas  uint64
}

type handler func(req *gomemcached.MCRequest, s *storage) *gomemcached.MCResponse

var handlers = map[gomemcached.CommandCode]handler{
	gomemcached.SET:    handleSet,
	gomemcached.GET:    handleGet,
	gomemcached.DELETE: handleDelete,
	gomemcached.FLUSH:  handleFlush,
}

func RunServer(input chan chanReq) {
	var s storage
	s.data = make(map[string]gomemcached.MCItem)
	for {
		req := <-input
		log.Printf("Got a request: %s", req.req)
		req.res <- dispatch(req.req, &s)
	}
}

func dispatch(req *gomemcached.MCRequest, s *storage) (rv *gomemcached.MCResponse) {
	if h, ok := handlers[req.Opcode]; ok {
		rv = h(req, s)
	} else {
		return notFound(req, s)
	}
	return
}

func notFound(req *gomemcached.MCRequest, s *storage) *gomemcached.MCResponse {
	var response gomemcached.MCResponse
	response.Status = gomemcached.UNKNOWN_COMMAND
	return &response
}

func handleSet(req *gomemcached.MCRequest, s *storage) (ret *gomemcached.MCResponse) {
	ret = &gomemcached.MCResponse{}
	var item gomemcached.MCItem

	item.Flags = binary.BigEndian.Uint32(req.Extras)
	item.Expiration = binary.BigEndian.Uint32(req.Extras[4:])
	item.Data = req.Body
	ret.Status = gomemcached.SUCCESS
	s.cas += 1
	item.Cas = s.cas
	ret.Cas = s.cas

	s.data[string(req.Key)] = item
	return
}

func handleGet(req *gomemcached.MCRequest, s *storage) (ret *gomemcached.MCResponse) {
	ret = &gomemcached.MCResponse{}
	if item, ok := s.data[string(req.Key)]; ok {
		ret.Status = gomemcached.SUCCESS
		ret.Extras = make([]byte, 4)
		binary.BigEndian.PutUint32(ret.Extras, item.Flags)
		ret.Cas = item.Cas
		ret.Body = item.Data
	} else {
		ret.Status = gomemcached.KEY_ENOENT
	}
	return
}

func handleFlush(req *gomemcached.MCRequest, s *storage) (ret *gomemcached.MCResponse) {
	ret = &gomemcached.MCResponse{}
	delay := binary.BigEndian.Uint32(req.Extras)
	if delay > 0 {
		log.Printf("Delay not supported (got %d)", delay)
	}
	s.data = make(map[string]gomemcached.MCItem)
	return
}

func handleDelete(req *gomemcached.MCRequest, s *storage) (ret *gomemcached.MCResponse) {
	ret = &gomemcached.MCResponse{}
	delete(s.data, string(req.Key))
	return
}
