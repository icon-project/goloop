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

public class MemberDecl {
    private final int access;
    private final Member member;

    public MemberDecl(int access, Member member) {
        this.access = access;
        this.member = member;
    }

    public MemberDecl(int access, String name, String descriptor) {
        this(access, new Member(name, descriptor));
    }

    public int getAccess() {
        return access;
    }

    public Member getMember() {
        return member;
    }

    public String getName() {
        return member.getName();
    }

    public String getDescriptor() {
        return member.getDescriptor();
    }
}
