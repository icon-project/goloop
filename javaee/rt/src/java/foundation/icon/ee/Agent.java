package foundation.icon.ee;

public interface Agent {
    ThreadLocal<Agent> agent = new ThreadLocal<>();

    static Agent get() {
        return agent.get();
    }

    boolean isClassMeteringEnabled();
}
