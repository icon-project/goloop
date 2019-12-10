package foundation.icon.ee.types;

import i.InternedClasses;

import java.util.List;

public class DAppRuntimeState {
    private List<Object> objects;
    private ObjectGraph graph;

    public DAppRuntimeState(List<Object> objects, ObjectGraph graph) {
        this.objects = objects;
        this.graph = graph;
    }

    public List<Object> getObjects() {
        return objects;
    }

    public ObjectGraph getGraph() {
        return graph;
    }
}
