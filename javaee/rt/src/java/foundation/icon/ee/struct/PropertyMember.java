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

import org.objectweb.asm.Type;

public class PropertyMember {
    private final int sort;
    private final Type declaringType;
    private final Member member;

    public static final int FIELD = 0;
    public static final int GETTER = 1;
    public static final int SETTER = 2;

    public PropertyMember(int sort, Type declaringType, String originalName,
            String descriptor) {
        this(sort, declaringType, originalName, Type.getType(descriptor));
    }

    public PropertyMember(int sort, Type declaringType, String originalName, Type type) {
        this.sort = sort;
        this.declaringType = declaringType;
        this.member = new Member(originalName, type);
    }

    public int getSort() {
        return sort;
    }

    public Type getDeclaringType() {
        return declaringType;
    }

    public String getOriginalName() {
        return member.getName();
    }

    public Type getOriginalType() {
        return member.getType();
    }

    public Type getType() {
        if (sort==FIELD) {
            return getOriginalType();
        } else if (sort==GETTER) {
            return getOriginalType().getReturnType();
        }
        assert sort==SETTER;
        return getOriginalType().getArgumentTypes()[0];
    }

    public String getName() {
        if (sort==FIELD) {
            return getOriginalName();
        }
        assert sort==GETTER || sort==SETTER;
        var pre = getOriginalName().startsWith("is") ? 2 : 3;
        return foundation.icon.ee.struct.Property.decapitalize(
                getOriginalName().substring(pre));
    }

    public Member getMember() {
        return member;
    }
}
