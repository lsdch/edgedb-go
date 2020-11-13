// This source file is part of the EdgeDB open source project.
//
// Copyright 2020-present EdgeDB Inc. and the EdgeDB authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package edgedb

import (
	"context"
	"fmt"
	"net"

	"github.com/edgedb/edgedb-go/protocol/buff"
	"github.com/edgedb/edgedb-go/protocol/message"
	"github.com/xdg/scram"
)

func connect(ctx context.Context, conn net.Conn, opts *Options) (err error) {
	buf := buff.NewWriter(nil)
	buf.BeginMessage(message.ClientHandshake)
	buf.PushUint16(0) // major version
	buf.PushUint16(8) // minor version
	buf.PushUint16(2) // number of parameters
	buf.PushString("database")
	buf.PushString(opts.Database)
	buf.PushString("user")
	buf.PushString(opts.User)
	buf.PushUint16(0) // no extensions
	buf.EndMessage()

	err = writeAndRead(ctx, conn, buf.Unwrap())
	if err != nil {
		return err
	}

	for buf.Next() {
		msg := buf.PopMessage()

		switch msg.Type {
		case message.ServerHandshake:
			// The client _MUST_ close the connection
			// if the protocol version can't be supported.
			// https://edgedb.com/docs/internals/protocol/overview
			major := msg.PopUint16()
			minor := msg.PopUint16()

			if major != 0 || minor != 8 {
				err = conn.Close()
				if err != nil {
					return err
				}

				err = fmt.Errorf(
					"unsupported protocol version: %v.%v",
					major,
					minor,
				)
				return err
			}
		case message.ServerKeyData:
			msg.Discard(32) // key data
		case message.ReadyForCommand:
			msg.PopUint16() // header count (assume 0)
			msg.PopUint8()  // transaction state
		case message.Authentication:
			if msg.PopUint32() == 0 { // auth status
				continue
			}

			// skip supported SASL methods
			n := int(msg.PopUint32()) // method count
			for i := 0; i < n; i++ {
				msg.PopBytes()
			}

			err := authenticate(ctx, conn, opts)
			if err != nil {
				return err
			}
		case message.ErrorResponse:
			return decodeError(msg)
		default:
			return fmt.Errorf("unexpected message type: 0x%x", msg.Type)
		}
		msg.Finish()
	}
	return nil
}

func authenticate(
	ctx context.Context,
	conn net.Conn,
	opts *Options,
) (err error) {
	client, err := scram.SHA256.NewClient(opts.User, opts.Password, "")
	if err != nil {
		panic(err)
	}

	conv := client.NewConversation()
	scramMsg, err := conv.Step("")
	if err != nil {
		panic(err)
	}

	buf := buff.NewWriter(nil)
	buf.BeginMessage(message.AuthenticationSASLInitialResponse)
	buf.PushString("SCRAM-SHA-256")
	buf.PushString(scramMsg)
	buf.EndMessage()

	err = writeAndRead(ctx, conn, buf.Unwrap())
	if err != nil {
		return err
	}

	msg := buf.PopMessage()
	switch msg.Type {
	case message.Authentication:
		authStatus := msg.PopUint32()
		if authStatus != 0xb {
			return fmt.Errorf(
				"unexpected authentication status: 0x%x",
				authStatus,
			)
		}

		scramRcv := msg.PopString()
		scramMsg, err = conv.Step(scramRcv)
		if err != nil {
			return err
		}
	case message.ErrorResponse:
		return decodeError(msg)
	default:
		return fmt.Errorf("unexpected message type: 0x%x", msg.Type)
	}
	msg.Finish()

	buf.Reset()
	buf.BeginMessage(message.AuthenticationSASLResponse)
	buf.PushString(scramMsg)
	buf.EndMessage()

	err = writeAndRead(ctx, conn, buf.Unwrap())
	if err != nil {
		return err
	}

	for buf.Next() {
		msg := buf.PopMessage()

		switch msg.Type {
		case message.Authentication:
			authStatus := msg.PopUint32()
			switch authStatus {
			case 0:
			case 0xc:
				scramRcv := msg.PopString()
				_, err = conv.Step(scramRcv)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf(
					"unexpected authentication status: 0x%x",
					authStatus,
				)
			}
		case message.ServerKeyData:
			msg.Discard(32) // key data
		case message.ReadyForCommand:
			msg.PopUint16() // header count (assume 0)
			msg.PopUint8()  // transaction state
		case message.ErrorResponse:
			return decodeError(msg)
		default:
			return fmt.Errorf("unexpected message type: 0x%x", msg.Type)
		}
		msg.Finish()
	}

	return nil
}
