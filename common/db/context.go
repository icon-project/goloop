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

func (f Flags) Get(n string) interface{} {
	if f == nil {
		return nil
	} else {
		return f[n]
	}
}

func (f Flags) Clone() Flags {
	flags := make(map[string]interface{})
	if len(f) > 0 {
		for k, v := range f {
			flags[k] = v
		}
	}
	return flags
}

func (f Flags) Merged(flags Flags) Flags {
	newFlags := f.Clone()
	if len(flags) > 0 {
		for k, v := range flags {
			newFlags[k] = v
		}
	}
	return newFlags
}

type ContextBuilder interface {
	WithFlags(flags Flags) Context
}

type Context interface {
	Database
	ContextBuilder
	GetFlag(n string) interface{}
	Flags() Flags
}

type databaseContext struct {
	Database
	flags Flags
}

func (c *databaseContext) WithFlags(flags Flags) Context {
	newFlags := c.flags.Merged(flags)
	return &databaseContext{c.Database, newFlags}
}

func (c *databaseContext) GetFlag(name string) interface{} {
	return c.flags.Get(name)
}

func (c *databaseContext) Flags() Flags {
	return c.flags.Clone()
}

func WithFlags(database Database, flags Flags) Context {
	if database == nil {
		return nil
	}
	if cb, ok := database.(ContextBuilder); ok {
		return cb.WithFlags(flags)
	} else {
		return &databaseContext{database, flags}
	}
}

func GetFlag(database Database, name string) interface{} {
	if ctx, ok := database.(Context); ok {
		return ctx.GetFlag(name)
	} else {
		return nil
	}
}
