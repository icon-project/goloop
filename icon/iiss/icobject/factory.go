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

type ImplFactory func(tag Tag) (Impl, error)

type databaseWithFactory struct {
	db.Database
	factory ImplFactory
}

func AttachObjectFactory(database db.Database, factory ImplFactory) db.Database {
	if dwf, ok := database.(*databaseWithFactory); ok {
		return dwf
	}
	return &databaseWithFactory{
		database, factory,
	}
}

func FactoryOf(database db.Database) ImplFactory {
	if dbf, ok := database.(*databaseWithFactory); ok {
		return dbf.factory
	}
	return nil
}
