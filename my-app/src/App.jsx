import React, { useEffect, useState, useCallback } from "react";
import ReactFlow, { MiniMap, Controls, Background, useNodesState, useEdgesState } from "reactflow";
import "reactflow/dist/style.css";
import dagre from "dagre";

//const SSE_URL = "http://10.51.52.152:8081/api/sse/resource_service/reservation/all/c013f9f0-d6b0-4b00-a720-ff627e15791d";
const SSE_URL = "http://localhost:8085/stream";
const graphLayout = new dagre.graphlib.Graph();
graphLayout.setDefaultEdgeLabel(() => ({}));

graphLayout.setGraph({ rankdir: "TB" }); // Top to Bottom layout

const getLayoutedElements = (nodes, edges) => {
  nodes.forEach((node) => graphLayout.setNode(node.id, { width: 150, height: 50 }));
  edges.forEach((edge) => graphLayout.setEdge(edge.source, edge.target));
  dagre.layout(graphLayout);
  return nodes.map((node) => {
    const { x, y } = graphLayout.node(node.id);
    return { ...node, position: { x, y } };
  });
};

const Graph = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [collapsedGroups, setCollapsedGroups] = useState({});

  useEffect(() => {
    const eventSource = new EventSource(SSE_URL);
    eventSource.onmessage = (event) => {
      console.log("event=%o", event)
      const eventJSON = JSON.parse(event.data)
      if (eventJSON.type === "state") {
      const data = eventJSON.state_data.resdata.dag;
      let updatedNodes = data.nodes.map((n) => ({ id: n.id, data: { label: n.label }, position: { x: 0, y: 0 } }));
      let updatedEdges = data.edges.map((e) => ({ id: e.id, source: e.source, target: e.target }));
      updatedNodes = getLayoutedElements(updatedNodes, updatedEdges);
      setNodes(updatedNodes);
      setEdges(updatedEdges);
      } else {
        console.log("Event type was %o ignoring", eventJSON.type)
      }
    };
    return () => eventSource.close();
  }, []);

  const toggleGroup = useCallback((groupId) => {
    setCollapsedGroups((prev) => ({ ...prev, [groupId]: !prev[groupId] }));
  }, []);

  return (
    <div style={{ width: "100vw", height: "100vh" }}>
      <ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} fitView>
        <MiniMap />
        <Controls />
        <Background />
      </ReactFlow>
    </div>
  );
};

export default Graph;
