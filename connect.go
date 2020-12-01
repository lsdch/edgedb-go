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

	"github.com/edgedb/edgedb-go/protocol/buff"
	"github.com/edgedb/edgedb-go/protocol/message"
	"github.com/xdg/scram"
)

func (c *baseConn) connect(ctx context.Context, cfg *connConfig) error {
	buf := buff.New(nil)
	buf.BeginMessage(message.ClientHandshake)
	buf.PushUint16(0) // major version
	buf.PushUint16(8) // minor version
	buf.PushUint16(2) // number of parameters
	buf.PushString("database")
	buf.PushString(cfg.database)
	buf.PushString("user")
	buf.PushString(cfg.user)
	buf.PushUint16(0) // no extensions
	buf.EndMessage()

	if err := c.writeAndRead(ctx, buf.Unwrap()); err != nil {
		return err
	}

	for buf.Next() {
		switch buf.MsgType {
		case message.ServerHandshake:
			// The client _MUST_ close the connection
			// if the protocol version can't be supported.
			// https://edgedb.com/docs/internals/protocol/overview
			major := buf.PopUint16()
			minor := buf.PopUint16()

			if major != 0 || minor != 8 {
				if err := c.conn.Close(); err != nil {
					return err
				}

				return fmt.Errorf(
					"unsupported protocol version: %v.%v",
					major,
					minor,
				)
			}
		case message.ServerKeyData:
			buf.Discard(32) // key data
		case message.ReadyForCommand:
			buf.PopUint16() // header count (assume 0)
			buf.PopUint8()  // transaction state
		case message.Authentication:
			if buf.PopUint32() == 0 { // auth status
				continue
			}

			// skip supported SASL methods
			n := int(buf.PopUint32()) // method count
			for i := 0; i < n; i++ {
				buf.PopBytes()
			}

			if err := c.authenticate(ctx, cfg); err != nil {
				return err
			}
		case message.ErrorResponse:
			return decodeError(buf)
		default:
			return fmt.Errorf("unexpected message type: 0x%x", buf.MsgType)
		}
	}
	return nil
}

func (c *baseConn) authenticate(ctx context.Context, cfg *connConfig) error {
	client, err := scram.SHA256.NewClient(cfg.user, cfg.password, "")
	if err != nil {
		return err
	}

	conv := client.NewConversation()
	scramMsg, err := conv.Step("")
	if err != nil {
		return err
	}

	buf := buff.New(nil)
	buf.BeginMessage(message.AuthenticationSASLInitialResponse)
	buf.PushString("SCRAM-SHA-256")
	buf.PushString(scramMsg)
	buf.EndMessage()

	err = c.writeAndRead(ctx, buf.Unwrap())
	if err != nil {
		return err
	}

	buf.Next()
	switch buf.MsgType {
	case message.Authentication:
		authStatus := buf.PopUint32()
		if authStatus != 0xb {
			return fmt.Errorf(
				"unexpected authentication status: 0x%x",
				authStatus,
			)
		}

		scramRcv := buf.PopString()
		scramMsg, err = conv.Step(scramRcv)
		if err != nil {
			return err
		}
	case message.ErrorResponse:
		return decodeError(buf)
	default:
		return fmt.Errorf("unexpected message type: 0x%x", buf.MsgType)
	}
	buf.Finish()

	buf.Reset()
	buf.BeginMessage(message.AuthenticationSASLResponse)
	buf.PushString(scramMsg)
	buf.EndMessage()

	err = c.writeAndRead(ctx, buf.Unwrap())
	if err != nil {
		return err
	}

	for buf.Next() {
		switch buf.MsgType {
		case message.Authentication:
			authStatus := buf.PopUint32()
			switch authStatus {
			case 0:
			case 0xc:
				scramRcv := buf.PopString()
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
			buf.Discard(32) // key data
		case message.ReadyForCommand:
			buf.PopUint16() // header count (assume 0)
			buf.PopUint8()  // transaction state
		case message.ErrorResponse:
			return decodeError(buf)
		default:
			return fmt.Errorf("unexpected message type: 0x%x", buf.MsgType)
		}
	}

	return nil
}

func (c *baseConn) terminate() error {
	// todo
	return nil
}
