/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"

	grpc2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/grpc"
	hash2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/protos"
)

var logger = flogging.MustGetLogger("view-sdk.client")

type TimeFunc func() time.Time

type SigningIdentity interface {
	Serialize() ([]byte, error)

	Sign(msg []byte) ([]byte, error)
}

//go:generate counterfeiter -o mock/view_peer_client.go -fake-name ViewServiceClient . ViewServiceClient

// ViewServiceClient defines an interface that creates a client to communicate with the view service in a peer
type ViewServiceClient interface {
	// CreateViewClient creates a grpc connection and client to view peer
	CreateViewClient() (*grpc.ClientConn, protos.ViewServiceClient, error)

	// Certificate returns tls client certificate
	Certificate() *tls.Certificate
}

// ViewServiceClientImpl implements ViewServiceClient interface
type ViewServiceClientImpl struct {
	Address            string
	ServerNameOverride string
	GRPCClient         *grpc2.GRPCClient
}

func (pc *ViewServiceClientImpl) CreateViewClient() (*grpc.ClientConn, protos.ViewServiceClient, error) {
	logger.Debugf("opening connection to [%s]", pc.Address)
	conn, err := pc.GRPCClient.NewConnection(pc.Address)
	if err != nil {
		logger.Errorf("failed creating connection to [%s]: [%s]", pc.Address, err)
		return conn, nil, errors.Wrapf(err, "failed creating connection to [%s]", pc.Address)
	}
	logger.Debugf("opening connection to [%s], done.", pc.Address)

	return conn, protos.NewViewServiceClient(conn), nil
}

func (pc *ViewServiceClientImpl) Certificate() *tls.Certificate {
	cert := pc.GRPCClient.Certificate()
	return &cert
}

// client implements network.Client interface
type client struct {
	Address           string
	ViewServiceClient ViewServiceClient
	RandomnessReader  io.Reader
	Time              TimeFunc
	SigningIdentity   SigningIdentity
	hasher            hash2.Hasher
}

func New(config *Config, sID SigningIdentity, hasher hash2.Hasher) (*client, error) {
	// create a grpc client for view peer
	grpcClient, err := grpc2.CreateGRPCClient(config.FSCNode)
	if err != nil {
		return nil, err
	}

	return &client{
		Address:          config.FSCNode.Address,
		RandomnessReader: rand.Reader,
		Time:             time.Now,
		ViewServiceClient: &ViewServiceClientImpl{
			Address:            config.FSCNode.Address,
			ServerNameOverride: config.FSCNode.ServerNameOverride,
			GRPCClient:         grpcClient,
		},
		SigningIdentity: sID,
		hasher:          hasher,
	}, nil
}

func (s *client) CallView(fid string, input []byte) (interface{}, error) {
	logger.Debugf("Calling view [%s] on input [%s]", fid, string(input))
	payload := &protos.Command_CallView{CallView: &protos.CallView{
		Fid:   fid,
		Input: input,
	}}
	sc, err := s.CreateSignedCommand(payload, s.SigningIdentity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating signed command for [%s,%s]", fid, string(input))
	}

	commandResp, err := s.processCommand(context.Background(), sc)
	if err != nil {
		return nil, errors.Wrapf(err, "failed process command for [%s,%s]", fid, string(input))
	}

	if commandResp.GetCallViewResponse() == nil {
		return nil, errors.New("expected initiate view response, got nothing")
	}
	return commandResp.GetCallViewResponse().GetResult(), nil
}

func (s *client) Initiate(fid string, in []byte) (string, error) {
	panic("implement me")
}

func (s *client) Track(cid string) string {
	panic("implement me")
}

func (s *client) IsTxFinal(txid string) error {
	logger.Debugf("Calling IsTxFinal on txid [%s]", txid)
	payload := &protos.Command_IsTxFinal{IsTxFinal: &protos.IsTxFinal{
		Txid: txid,
	}}
	sc, err := s.CreateSignedCommand(payload, s.SigningIdentity)
	if err != nil {
		logger.Errorf("failed creating signed command to ask for finality of tx [%s] at [%s]", txid, s.Address)
		return errors.Wrapf(err, "failed creating signed command to ask for finality of tx [%s] at [%s]", txid, s.Address)
	}

	logger.Debugf("Contact the server to ask if tx [%s] final at [%s]", txid, s.Address)
	commandResp, err := s.processCommand(context.Background(), sc)
	if err != nil {
		logger.Errorf("failed process command to ask for finality of tx [%s] at [%s]", txid, s.Address)
		return errors.Wrapf(err, "failed process command to ask for finality of tx [%s] at [%s]", txid, s.Address)
	}
	logger.Debugf("Contact the server to ask if tx [%s] final at [%s]. Done", txid, s.Address)

	if commandResp.GetIsTxFinalResponse() == nil {
		logger.Errorf("expected response, got nothing while asking for finality of tx [%s] at [%s]", txid, s.Address)
		return errors.Errorf("expected response, got nothing while asking for finality of tx [%s] at [%s]", txid, s.Address)
	}

	respPayload := commandResp.GetIsTxFinalResponse().GetPayload()
	logger.Debugf("Is tx [%s] final at [%s]? [%s]", txid, s.Address, string(respPayload))
	if len(respPayload) == 0 {
		return nil
	}
	return errors.New(string(respPayload))
}

func (s *client) IsHashFinal(hash []byte) (<-chan *protos.IsHashFinalResponse, error) {
	payload := &protos.Command_IsHashFinal{IsHashFinal: &protos.IsHashFinal{
		Hash: hash,
	}}
	sc, err := s.CreateSignedCommand(payload, s.SigningIdentity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating signed command to ask for finality of hash [%s] at [%s]", hash2.Hashable(hash).String(), s.Address)
	}

	scc, err := s.streamCommand(context.Background(), sc)
	if err != nil {
		return nil, errors.Wrapf(err, "failed process command to ask for finality of hash [%s] at [%s]", hash2.Hashable(hash).String(), s.Address)
	}

	receiver := make(chan *protos.IsHashFinalResponse, 10)
	go func(stream protos.ViewService_StreamCommandClient, receiver chan *protos.IsHashFinalResponse) {
		defer close(receiver)
		for {
			scr, err := stream.Recv()
			logger.Debugf("Is Hash [%s] final, got a message to parse [%s]", hash2.Hashable(hash).String(), err)
			if err == io.EOF {
				logger.Warnf("Is Hash [%s] final, stream EOF", hash2.Hashable(hash).String())
				break
			}
			if err != nil {
				logger.Errorf("Is Hash [%s] final, this should not happen [%s]", hash2.Hashable(hash).String(), err)
				break
			}

			commandResp := &protos.CommandResponse{}
			err = proto.Unmarshal(scr.Response, commandResp)
			if err != nil {
				logger.Errorf("failed to unmarshal command response [%s]", err)
				return
			}
			if commandResp.GetErr() != nil {
				logger.Errorf("Is Hash [%s] final, error from view during process command [%s]", commandResp.GetErr().GetMessage())
				return
			}

			logger.Debugf("Is Hash [%s] final, got response [%v]", commandResp.GetIsHashFinalResponse())
			receiver <- commandResp.GetIsHashFinalResponse()
		}
	}(scc, receiver)

	return receiver, nil
}

// processCommand calls view client to send grpc request and returns a CommandResponse
func (s *client) processCommand(ctx context.Context, sc *protos.SignedCommand) (*protos.CommandResponse, error) {
	logger.Debugf("get view service client...")
	conn, client, err := s.ViewServiceClient.CreateViewClient()
	logger.Debugf("get view service client...done")
	if conn != nil {
		logger.Debugf("get view service client...got a connection")
		defer conn.Close()
	}
	if err != nil {
		logger.Errorf("failed creating view client [%s]", err)
		return nil, errors.Wrap(err, "failed creating view client")
	}

	logger.Debugf("process command [%s]", sc.String())
	scr, err := client.ProcessCommand(ctx, sc)
	if err != nil {
		logger.Errorf("failed view client process command [%s]", err)
		return nil, errors.Wrap(err, "failed view client process command")
	}

	logger.Debugf("parse answer [%s]", hash2.Hashable(scr.Response).String())
	commandResp := &protos.CommandResponse{}
	err = proto.Unmarshal(scr.Response, commandResp)
	if err != nil {
		logger.Errorf("failed to unmarshal command response [%s]", err)
		return nil, errors.Wrapf(err, "failed to unmarshal command response")
	}
	if commandResp.GetErr() != nil {
		logger.Errorf("error from view during process command: %s", commandResp.GetErr().GetMessage())
		return nil, errors.Errorf("error from view during process command: %s", commandResp.GetErr().GetMessage())
	}

	logger.Debugf("process command [%s] done", sc.String())
	return commandResp, nil
}

func (s *client) streamCommand(ctx context.Context, sc *protos.SignedCommand, opts ...grpc.CallOption) (protos.ViewService_StreamCommandClient, error) {
	logger.Debugf("[stream] get view service client...")
	conn, client, err := s.ViewServiceClient.CreateViewClient()
	logger.Debugf("[stream] get view service client...done")
	if conn != nil {
		logger.Debugf("[stream] get view service client...got a connection")
		//defer conn.Close()
	}
	if err != nil {
		logger.Errorf("[stream] failed creating view client [%s]", err)
		return nil, errors.Wrap(err, "[stream] failed creating view client")
	}

	logger.Debugf("stream command [%s]", sc.String())
	scc, err := client.StreamCommand(ctx, sc, opts...)
	if err != nil {
		logger.Errorf("[stream] failed view client stream command [%s]", err)
		return nil, errors.Wrap(err, "[stream] failed view client stream command")
	}
	logger.Debugf("stream command [%s], done!", sc.String())

	return scc, nil
}

func (s *client) CreateSignedCommand(payload interface{}, signingIdentity SigningIdentity) (*protos.SignedCommand, error) {
	command, err := commandFromPayload(payload)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 32)
	_, err = io.ReadFull(s.RandomnessReader, nonce)
	if err != nil {
		return nil, err
	}

	ts, err := ptypes.TimestampProto(s.Time())
	if err != nil {
		return nil, err
	}

	creator, err := signingIdentity.Serialize()
	if err != nil {
		return nil, err
	}

	// check for client certificate and compute SHA2-256 on certificate if present
	tlsCertHash, err := grpc2.GetTLSCertHash(s.ViewServiceClient.Certificate(), s.hasher)
	if err != nil {
		return nil, err
	}
	command.Header = &protos.Header{
		Timestamp:   ts,
		Nonce:       nonce,
		Creator:     creator,
		TlsCertHash: tlsCertHash,
	}

	raw, err := proto.Marshal(command)
	if err != nil {
		return nil, err
	}

	signature, err := signingIdentity.Sign(raw)
	if err != nil {
		return nil, err
	}

	sc := &protos.SignedCommand{
		Command:   raw,
		Signature: signature,
	}
	return sc, nil
}

func commandFromPayload(payload interface{}) (*protos.Command, error) {
	switch t := payload.(type) {
	case *protos.Command_InitiateView:
		return &protos.Command{Payload: t}, nil
	case *protos.Command_TrackView:
		return &protos.Command{Payload: t}, nil
	case *protos.Command_CallView:
		return &protos.Command{Payload: t}, nil
	case *protos.Command_IsTxFinal:
		return &protos.Command{Payload: t}, nil
	case *protos.Command_IsHashFinal:
		return &protos.Command{Payload: t}, nil
	default:
		return nil, errors.Errorf("command type not recognized: %T", t)
	}
}
