package foundation.icon.icx.transport.monitor;

public interface Monitor<T> {
    interface Listener<T> {
        void onStart();
        void onEvent(T msg);
        void onError(long code);
        void onClose();
    }

    /**
     *
     *
     * @param listener
     * @return
     */
    boolean start(Listener<T> listener);

    /**
     *
     */
    void stop();
}
