import React, { useState, useEffect } from 'react';
import { Search, Database, Brain, Clock, Loader2, Terminal, ChevronDown, ChevronRight, Table, FileJson } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

const SmartInsights = () => {
    const [configs, setConfigs] = useState({
        database: [],
        llm: []
    });
    const [selectedDB, setSelectedDB] = useState('');
    const [selectedLLM, setSelectedLLM] = useState({ provider: '', name: '' });
    const [question, setQuestion] = useState('');
    const [currentQuery, setCurrentQuery] = useState(null);
    const [responses, setResponses] = useState([]);
    const [isLoading, setIsLoading] = useState(false);
    const [showLogs, setShowLogs] = useState(true);

    useEffect(() => {
        const fetchConfigs = async () => {
            try {
                const [dbResponse, llmResponse] = await Promise.all([
                    fetch('/databases'),
                    fetch('/llm')
                ]);
                const dbData = await dbResponse.json() || [];
                const llmData = await llmResponse.json() || [];

                // Transform array of LLM configs into array with provider info
                const llmConfigs = llmData.map(config => ({
                    ...config,
                    provider: config.type // Use the type as provider
                }));

                setConfigs({
                    database: dbData,
                    llm: llmConfigs
                });

                // Auto-select first config if available
                if (dbData.length > 0) setSelectedDB(dbData[0].name);

                // Modify the LLM selection logic
                if (llmConfigs.length > 0) {
                    setSelectedLLM({
                        provider: llmConfigs[0].type,
                        name: llmConfigs[0].name
                    });
                }
            } catch (error) {
                console.error('Failed to fetch configurations:', error);
            }
        };
        fetchConfigs();
    }, []);

    useEffect(() => {
        let pollInterval;
        if (currentQuery && currentQuery.status !== 'completed') {
            pollInterval = setInterval(async () => {
                try {
                    const response = await fetch(`/assistant/ask/${currentQuery.uuid}`);
                    const data = await response.json();

                    setResponses(prevResponses => {
                        return prevResponses.map(resp => {
                            if (resp.uuid === data.uuid) {
                                if (resp.response?.length >= data.response?.length) {
                                    return resp;
                                }
                                return data;
                            }
                            return resp;
                        });
                    });

                    if (data.status === 'completed') {
                        setCurrentQuery(null);
                        setIsLoading(false);
                        setShowLogs(false);
                        clearInterval(pollInterval);
                    } else {
                        setCurrentQuery(data);
                    }
                } catch (error) {
                    console.error('Polling error:', error);
                    clearInterval(pollInterval);
                    setIsLoading(false);
                }
            }, 2000);
        }
        return () => clearInterval(pollInterval);
    }, [currentQuery]);

    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!selectedDB || !selectedLLM.provider || !selectedLLM.name || !question.trim()) return;

        setIsLoading(true);
        setShowLogs(true);
        try {
            const response = await fetch('/assistant/ask', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    db_configuration_name: selectedDB,
                    question: question.trim(),
                    options: {
                        llm_provider: selectedLLM.provider,
                        llm_config: selectedLLM.name
                    }
                })
            });
            const data = await response.json();
            setCurrentQuery(data);
            setResponses(prev => [data, ...prev]);
            setQuestion('');
        } catch (error) {
            console.error('Failed to submit question:', error);
            setIsLoading(false);
        }
    };

    const handleLLMChange = (e) => {
        const [provider, name] = e.target.value.split('|');
        setSelectedLLM({ provider, name });
    };

    const getFinalResponse = (response) => {
        if (!response.response) return null;
        return response.response.find(update => update.type === 'final_response');
    };

    const getProgressUpdates = (response) => {
        if (!response.response) return [];
        return response.response.filter(update => update.type !== 'final_response');
    };

    const isMarkdownResponse = (response) => {
        return response.text.includes('#') ||
            response.text.includes('```') ||
            response.text.includes('*') ||
            response.text.includes('|') ||
            response.text.includes('- ');
    };

    const renderResponseContent = (response) => {
        if (isMarkdownResponse(response)) {
            return (
                <div className="prose prose-sm max-w-none prose-table:table-auto prose-td:border prose-td:p-2 prose-th:border prose-th:p-2">
                    <ReactMarkdown
                        remarkPlugins={[remarkGfm]}
                        components={{
                            table: ({node, ...props}) => (
                                <table className="border-collapse border w-full" {...props} />
                            ),
                            th: ({node, ...props}) => (
                                <th className="bg-gray-100 border-gray-300 text-left p-2" {...props} />
                            ),
                            td: ({node, ...props}) => (
                                <td className="border-gray-300 p-2" {...props} />
                            ),
                            code: ({node, inline, ...props}) => (
                                inline ?
                                    <code className="bg-gray-100 px-1 py-0.5 rounded" {...props} /> :
                                    <code className="block bg-gray-100 p-4 rounded-lg" {...props} />
                            )
                        }}
                    >
                        {response.text}
                    </ReactMarkdown>
                </div>
            );
        }
        return (
            <pre className="text-sm text-gray-700 whitespace-pre-wrap">
                {response.text}
            </pre>
        );
    };

    if (configs.database.length === 0 || configs.llm.length === 0) {
        return (
            <div className="text-center py-12">
                <Database className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 mb-2">No Configurations Found</h3>
                <p className="text-gray-500 mb-4">Please add database and LLM configurations before using the chat.</p>
                <button
                    onClick={() => window.location.hash = '#config'}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                >
                    Go to Configurations
                </button>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm border p-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                    <div>
                        <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-2">
                            <Database className="h-4 w-4" /> Database Configuration
                        </label>
                        <select
                            value={selectedDB}
                            onChange={(e) => setSelectedDB(e.target.value)}
                            className="w-full rounded-md border border-gray-300 p-2 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                        >
                            <option value="">Select Database</option>
                            {configs.database.map(config => (
                                <option key={config.name} value={config.name}>
                                    {config.name} ({config.type})
                                </option>
                            ))}
                        </select>
                    </div>
                    <div>
                        <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-2">
                            <Brain className="h-4 w-4" /> LLM Configuration
                        </label>
                        <select
                            value={`${selectedLLM.provider}|${selectedLLM.name}`}
                            onChange={handleLLMChange}
                            className="w-full rounded-md border border-gray-300 p-2 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                        >
                            <option value="">Select LLM</option>
                            {configs.llm.map(config => (
                                <option
                                    key={`${config.provider}|${config.name}`}
                                    value={`${config.provider}|${config.name}`}
                                >
                                    {config.name} ({config.provider})
                                </option>
                            ))}
                        </select>
                    </div>
                </div>
                <div className="flex flex-col md:flex-row gap-4">
                    <input
                        type="text"
                        value={question}
                        onChange={(e) => setQuestion(e.target.value)}
                        placeholder="Ask a question about your data..."
                        className="flex-1 rounded-md border border-gray-300 p-3 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                    />
                    <button
                        type="submit"
                        disabled={isLoading || !selectedDB || !selectedLLM.name || !question.trim()}
                        className="bg-blue-600 text-white px-6 py-3 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center min-w-[100px]"
                    >
                        {isLoading ? <Loader2 className="h-5 w-5 animate-spin" /> : 'Ask'}
                    </button>
                </div>
            </form>

            {responses.length > 0 && (
                <div className="mb-8">
                    <button
                        onClick={() => setShowLogs(!showLogs)}
                        className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-2 hover:text-blue-600"
                    >
                        {showLogs ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                        <Terminal className="h-4 w-4" />
                        Process Logs
                    </button>

                    {showLogs && (
                        <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-300 max-h-60 overflow-y-auto">
                            {responses.map((response) => (
                                <div key={response.uuid}>
                                    {getProgressUpdates(response).map((update, idx) => (
                                        <div key={`${response.uuid}-${idx}`} className="mb-2">
                                            <span className="text-green-400">{new Date(update.timestamp).toLocaleTimeString()}</span>
                                            <span className="text-blue-400"> [{update.type}]</span>
                                            <span className="text-gray-300"> {update.text}</span>
                                        </div>
                                    ))}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            <div className="space-y-6">
                {responses.map((response) => {
                    const finalResponse = getFinalResponse(response);
                    if (!finalResponse) return null;

                    return (
                        <div key={response.uuid} className="bg-white rounded-lg shadow-sm border overflow-hidden">
                            <div className="border-b bg-gray-50 p-4">
                                <div className="flex items-center justify-between">
                                    <h3 className="font-medium text-gray-900 flex items-center gap-2">
                                        <Search className="h-4 w-4 text-blue-600" />
                                        {response.question}
                                    </h3>
                                    <div className="flex items-center gap-2 text-sm text-gray-500">
                                        <Clock className="h-4 w-4" />
                                        {new Date(finalResponse.timestamp).toLocaleTimeString()}
                                    </div>
                                </div>
                            </div>

                            <div className="p-6">
                                <div className="flex items-center gap-2 mb-4">
                                    {finalResponse.type === 'table' ? (
                                        <Table className="h-5 w-5 text-blue-600" />
                                    ) : (
                                        <FileJson className="h-5 w-5 text-blue-600" />
                                    )}
                                    <span className="font-medium text-gray-700">Query Result</span>
                                </div>

                                <div className="bg-gray-50 rounded-lg p-4 overflow-x-auto">
                                    {renderResponseContent(finalResponse)}
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
};

export default SmartInsights;