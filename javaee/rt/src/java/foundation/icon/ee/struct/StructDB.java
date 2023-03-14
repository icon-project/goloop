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

import foundation.icon.ee.score.EEPType;
import foundation.icon.ee.types.Method;
import org.objectweb.asm.Type;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class StructDB {
    private final Map<Type, ClassPropertyMemberInfo> cpiMap;

    // readable struct to properties map including parent class properties.
    private final Map<Type, List<PropertyMember>> rPropsMap = new HashMap<>();

    // writable struct to properties map including parent class properties.
    private final Map<Type, List<PropertyMember>> wPropsMap = new HashMap<>();

    // TODO: separate
    private final Set<Type> parameterStructs = new HashSet<>();
    private final Set<Type> returnStructs = new HashSet<>();

    public StructDB(Map<String, byte[]> classMap) {
        this(classMap, false);
    }

    public StructDB(Map<String, byte[]> classMap, boolean onlyPublicClass) {
        this.cpiMap = ClassPropertyMemberInfo.map(classMap, onlyPublicClass);
        updatePropsMap();
    }

    private void requireNoCyclicInheritance(ClassPropertyMemberInfo cpi) {
        List<Type> visited = new ArrayList<>();
        while (cpi != null) {
            if (visited.contains(cpi.getType())) {
                throw new IllegalArgumentException("cyclic inheritance "
                        + cpi.getType().getClassName());
            }
            visited.add(cpi.getType());
            cpi = cpiMap.get(cpi.getSuperType());
        }
    }

    private void uniteByName(List<PropertyMember> list, List<PropertyMember> r) {
        next:
        for (var sp : r) {
            for (var wp : list) {
                if (sp.getName().equals(wp.getName())) {
                    continue next;
                }
            }
            list.add(sp);
        }
    }

    private List<PropertyMember> getRProps(ClassPropertyMemberInfo cpi) {
        List<PropertyMember> rProps = new ArrayList<>();
        var cur = cpi;
        while (cur != null) {
            uniteByName(rProps, cur.getGetters());
            cur = cpiMap.get(cur.getSuperType());
        }
        cur = cpi;
        while (cur != null) {
            uniteByName(rProps, cur.getFields());
            cur = cpiMap.get(cur.getSuperType());
        }
        if (rProps.isEmpty()) {
            return null;
        }
        return rProps;
    }

    private List<PropertyMember> getWProps(ClassPropertyMemberInfo cpi) {
        if (!cpi.isCreatable()) {
            return null;
        }
        List<PropertyMember> wProps = new ArrayList<>();
        var cur = cpi;
        while (cur != null) {
            nextSetter:
            for (var sp : cur.getSetters()) {
                for (var wp : wProps) {
                    if (sp.getName().equals(wp.getName())) {
                        if (!sp.getType().equals(wp.getType())) {
                            // conflicting. do not add to map
                            return null;
                        }
                        continue nextSetter;
                    }
                }
                wProps.add(sp);
            }
            cur = cpiMap.get(cur.getSuperType());
        }
        cur = cpi;
        while (cur != null) {
            uniteByName(wProps, cur.getFields());
            cur = cpiMap.get(cur.getSuperType());
        }
        if (wProps.isEmpty()) {
            return null;
        }
        return wProps;
    }

    private boolean isValidStruct(Type t, Map<Type, List<PropertyMember>> propsMap) {
        try {
            new TypeDetailCreator(propsMap).getTypeDetail(t);
        } catch (IllegalArgumentException e) {
            return false;
        }
        return true;
    }

    private void updatePropsMap() {
        var lwPropsMap = new HashMap<Type, List<PropertyMember>>();
        var lrPropsMap = new HashMap<Type, List<PropertyMember>>();
        for (var ce : cpiMap.entrySet()) {
            var c = ce.getValue();
            requireNoCyclicInheritance(c);

            if (c.isCreatable()) {
                var props = getWProps(c);
                if (props != null) {
                    lwPropsMap.put(c.getType(), props);
                }
            }
            var props = getRProps(c);
            if (props != null) {
                lrPropsMap.put(c.getType(), props);
            }
        }
        for (var wpe : lwPropsMap.entrySet()) {
            if (isValidStruct(wpe.getKey(), lwPropsMap)) {
                wPropsMap.put(wpe.getKey(), wpe.getValue());
            }
        }
        for (var rpe : lrPropsMap.entrySet()) {
            if (isValidStruct(rpe.getKey(), lrPropsMap)) {
                rPropsMap.put(rpe.getKey(), rpe.getValue());
            }
        }
    }

    public boolean isWritableStruct(Type t) {
        return wPropsMap.containsKey(t);
    }

    public boolean isReadableStruct(Type t) {
        return rPropsMap.containsKey(t);
    }

    private static class TypeDetailCreator {
        private final Map<Type, List<PropertyMember>> propsMap;
        private final List<Type> visiting = new ArrayList<>();

        public TypeDetailCreator(Map<Type, List<PropertyMember>> propsMap) {
            this.propsMap = propsMap;
        }

        Method.TypeDetail getTypeDetail(Type t) {
            if (t.getSort() == Type.ARRAY) {
                var elemType = t.getElementType();
                Method.TypeDetail elemTD;
                int elemEEPType;
                if (elemType.getSort() == Type.BYTE) {
                    elemTD = new Method.TypeDetail(Method.DataType.BYTES);
                    elemEEPType = ((t.getDimensions()-1) << Method.DataType.DIMENSION_SHIFT)
                            | elemTD.getType();
                } else {
                    elemTD = getTypeDetail(elemType);
                    elemEEPType = (t.getDimensions() << Method.DataType.DIMENSION_SHIFT)
                                    | elemTD.getType();
                }
                return new Method.TypeDetail(elemEEPType,
                        elemTD.getStructFields());
            }
            var eepType = EEPType.fromBasicFieldType.get(t);
            if (eepType != null) {
                return new Method.TypeDetail(eepType);
            }
            var props = propsMap.get(t);
            if (props == null) {
                throw new IllegalArgumentException("invalid t "
                        + t.getClassName());
            }
            if (visiting.contains(t)) {
                throw new IllegalArgumentException("cyclic property in "
                        + t.getClassName());
            }
            visiting.add(t);
            var fields = new ArrayList<Method.Field>();
            for (var p : props) {
                var ftd = getTypeDetail(p.getType());
                fields.add(new Method.Field(p.getName(), ftd.getType(),
                        ftd.getStructFields()));
            }
            visiting.remove(visiting.size() - 1);
            return new Method.TypeDetail(Method.DataType.STRUCT,
                    fields.toArray(new Method.Field[0]));
        }
    }

    private static class ReferredStructCollector {
        private final Map<Type, List<PropertyMember>> propsMap;

        public ReferredStructCollector(Map<Type, List<PropertyMember>> propsMap) {
            this.propsMap = propsMap;
        }

        void visit(Set<Type> rs, Type t) {
            if (t.getSort() == Type.ARRAY) {
                visit(rs, t.getElementType());
            }
            var props = propsMap.get(t);
            if (props == null) {
                return;
            }
            rs.add(t);
            for (var p : props) {
                visit(rs, p.getType());
            }
        }

        Set<Type> getReferredStructs(Type t) {
            Set<Type> rs = new HashSet<>();
            visit(rs, t);
            return rs;
        }
    }

    public List<PropertyMember> getReadableProperties(Type type) {
        return rPropsMap.get(type);
    }

    public List<PropertyMember> getWritableProperties(Type type) {
        return wPropsMap.get(type);
    }

    public boolean isValidParamTypeElement(Type type) {
        return EEPType.fromBasicParamType.containsKey(type)
                || isWritableStruct(type);
    }

    public boolean isValidParamType(Type type) {
        if (type.getSort() == Type.ARRAY) {
            return isValidParamTypeElement(type.getElementType());
        }
        return isValidParamTypeElement(type);
    }

    public boolean isValidReturnTypeElement(Type type) {
        return EEPType.fromBasicReturnType.containsKey(type)
                || isReadableStruct(type);
    }

    public boolean isValidReturnType(Type type) {
        if (type.getSort() == Type.ARRAY) {
            return isValidReturnTypeElement(type.getElementType());
        }
        return isValidReturnTypeElement(type);
    }

    public Method.TypeDetail getDetailFromParameterType(Type t) {
        var eepType = EEPType.fromBasicParamType.get(t);
        if (eepType != null) {
            return new Method.TypeDetail(eepType);
        }
        return new TypeDetailCreator(wPropsMap).getTypeDetail(t);
    }

    public void addParameterType(Type t) {
        var rtc = new ReferredStructCollector(wPropsMap);
        parameterStructs.addAll(rtc.getReferredStructs(t));
    }

    public Set<Type> getParameterStructs() {
        return parameterStructs;
    }

    public int getEEPTypeFromReturnType(Type t) {
        // handles byte[] here
        Integer eepType = EEPType.fromBasicReturnType.get(t);
        if (eepType != null) {
            return eepType;
        }
        if (t.getSort() == Type.ARRAY
                && isValidReturnTypeElement(t.getElementType())) {
            return Method.DataType.LIST;
        }
        if (isReadableStruct(t)) {
            return Method.DataType.DICT;
        }
        throw new IllegalArgumentException();
    }

    public void addReturnType(Type t) {
        var rtc = new ReferredStructCollector(rPropsMap);
        returnStructs.addAll(rtc.getReferredStructs(t));
    }

    public Set<Type> getReturnStructs() {
        return returnStructs;
    }
}
