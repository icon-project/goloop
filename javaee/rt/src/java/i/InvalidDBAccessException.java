package i;

import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.SystemException;

public class InvalidDBAccessException extends SystemException {
    public InvalidDBAccessException() {
        super(Status.InvalidContainerAccess);
    }

    public InvalidDBAccessException(String msg) {
        super(Status.InvalidContainerAccess, msg);
    }

    public InvalidDBAccessException(String message, Throwable cause) {
        super(Status.InvalidContainerAccess, message, cause);
    }

    public InvalidDBAccessException(Throwable cause) {
        super(Status.InvalidContainerAccess, cause);
    }
}
