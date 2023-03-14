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

package foundation.icon.ee.struct;

import foundation.icon.ee.util.ASM;
import org.objectweb.asm.Type;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ClassPropertyMemberInfo {
    private final Type type;
    private final Type superType;
    private final boolean isCreatable;
    private final List<PropertyMember> fields;
    private final List<PropertyMember> getters;
    private final List<PropertyMember> setters;

    public ClassPropertyMemberInfo(Type type, Type superType, boolean isCreatable,
            List<PropertyMember> fields, List<PropertyMember> getters,
            List<PropertyMember> setters) {
        this.type = type;
        this.superType = superType;
        this.isCreatable = isCreatable;
        this.fields = fields;
        this.getters = getters;
        this.setters = setters;
    }

    public Type getType() {
        return type;
    }

    public Type getSuperType() {
        return superType;
    }

    public boolean isCreatable() {
        return isCreatable;
    }

    public List<PropertyMember> getFields() {
        return fields;
    }

    public List<PropertyMember> getGetters() {
        return getters;
    }

    public List<PropertyMember> getSetters() {
        return setters;
    }

    public static ClassPropertyMemberInfo fromBytes(
            byte[] classBytes, boolean onlyPublicClass
    ) {
        return ASM.accept(classBytes, new ClassPropertyMemberInfoCollector(onlyPublicClass))
                .getClassPropertyInfo();
    }

    public static Map<Type, ClassPropertyMemberInfo> map(
            Map<String, byte[]> classMap, boolean onlyPublicClass) {
        var cpiMap = new HashMap<Type, ClassPropertyMemberInfo>();
        for (var e : classMap.entrySet()) {
            var cpi = ClassPropertyMemberInfo.fromBytes(e.getValue(), onlyPublicClass);
            cpiMap.put(cpi.getType(), cpi);
        }
        return cpiMap;
    }
}
