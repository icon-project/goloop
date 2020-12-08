/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package db

type Flags map[string]interface{}
type Context struct {
	Database
	info map[string]interface{}
}

func (c *Context) WithFlags(flags map[string]interface{}) *Context {
	if c == nil {
		return nil
	}
	info := make(map[string]interface{})
	if len(c.info) > 0 {
		for k, v := range c.info {
			info[k] = v
		}
	}
	if len(flags) > 0 {
		for k, v := range flags {
			info[k] = v
		}
	}
	return &Context{
		Database: c.Database,
		info:     info,
	}
}

func (c *Context) GetFlag(n string) interface{} {
	if c == nil {
		return nil
	}
	return c.info[n]
}

func WithFlags(dbase Database, flags Flags) Database {
	ctx := ContextOf(dbase)
	return ctx.WithFlags(flags)
}

func ContextOf(dbase Database) *Context {
	if dbase == nil {
		return nil
	}
	if dbc, ok := dbase.(*Context); ok {
		return dbc
	} else {
		return &Context{
			Database: dbase,
			info:     make(map[string]interface{}),
		}
	}
}

func GetFlag(db Database, name string) interface{} {
	return ContextOf(db).GetFlag(name)
}
