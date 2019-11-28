package foundation.icon.ee.types;

import i.InternedClasses;

import java.util.List;

public class DAppRuntimeState {
    private List<Object> objects;
    private InternedClasses internedClasses;
    private ObjectGraph graph;

    public DAppRuntimeState(List<Object> objects, InternedClasses internedClasses, ObjectGraph graph) {
        this.objects = objects;
        this.internedClasses = internedClasses;
        this.graph = graph;
    }

    public List<Object> getObjects() {
        return objects;
    }

    public InternedClasses getInternedClasses() {
        return internedClasses;
    }

    public ObjectGraph getGraph() {
        return graph;
    }
}
