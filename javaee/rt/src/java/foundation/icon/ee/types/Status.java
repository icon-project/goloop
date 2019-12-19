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

    public static final int UserReversionStart = 32;
    public static final int UserReversionEnd = 1000;
}
