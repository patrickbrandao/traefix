package main

import (
	"net"
	"testing"

	grpcInfopage "github.com/patrickbrandao/traefix/infopage/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGRPCInfopage(t *testing.T) {
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := grpc.NewServer()
	grpcInfopage.RegisterInfopageServer(s, infopageServer{})
	go func() { _ = s.Serve(l) }()
	defer s.Stop()
	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cc.Close() }()
	r, err := grpcInfopage.NewInfopageClient(cc).Infopage(t.Context(), &grpcInfopage.InfopageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if r.GetHostname() == "" {
		t.Fatal("empty hostname")
	}
}
