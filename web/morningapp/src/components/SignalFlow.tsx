type SignalNode = {
    label: string;
    arrowAccent?: "magenta" | "cyan";
};

const signalNodes: SignalNode[] = [
    {
        label: "WHOOP",
        arrowAccent: "magenta",
    },
    {
        label: "Coach",
        arrowAccent: "magenta",
    },
    {
        label: "Receipt",
        arrowAccent: "cyan",
    },
    {
        label: "Mac mini",
        arrowAccent: "cyan",
    },
    {
        label: "SP712",
    },
];

function SignalFlow() {
    return (
        <section className="panel panel-cyan signal-panel">
            <p className="terminal-label signal-panel__label">
                Signal flow
            </p>

            <div
                className="signal-flow"
                aria-label="WHOOP data processing and printing flow"
            >
                {signalNodes.map((node, index) => (
                    <SignalFlowNode
                        key={node.label}
                        node={node}
                        isLast={index === signalNodes.length - 1}
                    />
                ))}
            </div>
        </section>
    );
}

type SignalFlowNodeProps = {
    node: SignalNode;
    isLast: boolean;
};

function SignalFlowNode({
                            node,
                            isLast,
                        }: SignalFlowNodeProps) {
    return (
        <>
      <span className="signal-flow__node">
        {node.label}
      </span>

            {!isLast && node.arrowAccent ? (
                <span
                    className={[
                        "signal-flow__arrow",
                        `signal-flow__arrow--${node.arrowAccent}`,
                    ].join(" ")}
                    aria-hidden="true"
                >
          →
        </span>
            ) : null}
        </>
    );
}

export default SignalFlow;