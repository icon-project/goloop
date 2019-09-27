package org.aion.avm;

import i.PackageConstants;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class ArrayClassNameMapper {
    static private final Map<String, String> ORIGINAL_TO_CLASS_WRAPPER_MAP = Collections.unmodifiableMap(initializeOriginalNameToClassWrapperMap());
    static private final Map<String, String> CLASS_WRAPPER_TO_ORIGINAL_MAP = Collections.unmodifiableMap(initializeClassWrapperToOriginalMap());

    static private final Map<String, String> INTERFACE_WRAPPER_MAP = Collections.unmodifiableMap(initializeInterfacedWrapperMaps());

    private static HashMap<String, String> initializeClassWrapperToOriginalMap() {
        HashMap<String, String> classWrapperMap = new HashMap<>();
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "IntArray", "[I");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "ByteArray", "[B");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "BooleanArray", "[Z");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "CharArray", "[C");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "FloatArray", "[F");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "ShortArray", "[S");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "LongArray", "[J");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "DoubleArray", "[D");
        classWrapperMap.put(PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray", "[Ljava/lang/Object");
        return classWrapperMap;
    }

    private static HashMap<String, String> initializeInterfacedWrapperMaps(){
        HashMap<String, String> interfaceWrapperMap = new HashMap<>();
        interfaceWrapperMap.put("[L" + PackageConstants.kInternalSlashPrefix + "IObject", PackageConstants.kInternalSlashPrefix + "IObjectArray");
        interfaceWrapperMap.put("L" + PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray", PackageConstants.kInternalSlashPrefix + "IObjectArray");
        interfaceWrapperMap.put("[L" + PackageConstants.kShadowSlashPrefix + "java/lang/Object", PackageConstants.kInternalSlashPrefix + "IObjectArray");
        return interfaceWrapperMap;
    }

    private static HashMap<String, String> initializeOriginalNameToClassWrapperMap(){
        HashMap<String, String> wrapperToOriginalMap = initializeClassWrapperToOriginalMap();
        HashMap<String, String> reverseMap = new HashMap<>();
        for (Map.Entry<String, String> entry :wrapperToOriginalMap.entrySet()) {
            reverseMap.put(entry.getValue(), entry.getKey());
        }
        // add additional ObjectArray cases to the map
        reverseMap.put("[L" + PackageConstants.kShadowSlashPrefix + "java/lang/Object", PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray");
        reverseMap.put("[L" + PackageConstants.kInternalSlashPrefix + "IObject", PackageConstants.kArrayWrapperSlashPrefix + "ObjectArray");
        return reverseMap;
    }

    public static String getClassWrapper(String desc) {
        return ORIGINAL_TO_CLASS_WRAPPER_MAP.get(desc);
    }

    public static String getInterfaceWrapper(String desc) {
        return INTERFACE_WRAPPER_MAP.get(desc);
    }

    public static String getOriginalNameFromWrapper(String wrapperClassName) {
        return CLASS_WRAPPER_TO_ORIGINAL_MAP.get(wrapperClassName);
    }

}
