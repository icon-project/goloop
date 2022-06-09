/*
 * Copyright 2019 ICON Foundation
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

package foundation.icon.ee.types;

public class Status {
    public static final int Success = 0;
    public static final int UnknownFailure = 1;
    public static final int ContractNotFound = 2;
    public static final int MethodNotFound = 3;
    public static final int MethodNotPayable = 4;
    public static final int IllegalFormat = 5;
    public static final int InvalidParameter = 6;
    public static final int InvalidInstance = 7;
    public static final int InvalidContainerAccess = 8;
    public static final int AccessDenied = 9;
    public static final int OutOfStep = 10;
    public static final int OutOfBalance = 11;
    public static final int Timeout = 12;
    public static final int StackOverflow = 13;
    public static final int SkipTransaction = 14;
    public static final int PackageError = 15;
    public static final int IllegalObjectGraph = 16;

    public static final int UserReversionStart = 32;
    public static final int UserReversionEnd = 1000;

    public final static int FlagShift = 24;
    public final static int FlagMask = 0xFF000000;
    public final static int CodeMask = 0x00FFFFFF;

    public static final int FlagRerun = 0x01000000;

    public static int fromUserCode(int code) {
        code = code + UserReversionStart;
        return Math.max(UserReversionStart, Math.min(UserReversionEnd-1, code));
    }

    public static String getMessage(int status) {
        switch (status) {
            case Success:
                return "Success";
            case UnknownFailure:
                return "UnknownFailure";
            case ContractNotFound:
                return "ContractNotFound";
            case MethodNotFound:
                return "MethodNotFound";
            case MethodNotPayable:
                return "MethodNotPayable";
            case IllegalFormat:
                return "IllegalFormat";
            case InvalidParameter:
                return "InvalidParameter";
            case InvalidInstance:
                return "InvalidInstance";
            case InvalidContainerAccess:
                return "InvalidContainerAccess";
            case AccessDenied:
                return "AccessDenied";
            case OutOfStep:
                return "OutOfStep";
            case OutOfBalance:
                return "OutOfBalance";
            case Timeout:
                return "Timeout";
            case StackOverflow:
                return "StackOverflow";
            case SkipTransaction:
                return "SkipTransaction";
            case PackageError:
                return "PackageError";
            case IllegalObjectGraph:
                return "IllegalObjectGraph";
        }
        if (status >= UserReversionStart) {
            return String.format("Reverted(%d)", status - UserReversionStart);
        } else {
            return String.format("Unknown(code=%d)", status);
        }
    }
}
