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

package icobject

import "github.com/icon-project/goloop/common/db"

const (
	objectFactoryName = "objectFactory"
)

type ImplFactory func(tag Tag) (Impl, error)

func AttachObjectFactory(database db.Database, factory ImplFactory) db.Database {
	return db.WithFlags(database, db.Flags{
		objectFactoryName: factory,
	})
}

func FactoryOf(database db.Database) ImplFactory {
	flag := db.GetFlag(database, objectFactoryName)
	if factory, ok := flag.(ImplFactory); ok {
		return factory
	} else {
		return nil
	}
}
