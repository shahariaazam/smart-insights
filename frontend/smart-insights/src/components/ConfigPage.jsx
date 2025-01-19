import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Database, Brain } from 'lucide-react';

const ConfigPage = () => {
    const [activeSection, setActiveSection] = useState('database');
    const [configs, setConfigs] = useState({
        database: [],
        llm: []
    });
    const [isLoading, setIsLoading] = useState(false);
    const [showAddForm, setShowAddForm] = useState(false);
    const [formData, setFormData] = useState({
        name: '',
        type: '',
        host: '',
        port: '',
        db_name: '',
        username: '',
        password: '',
        api_key: '',
        model: '',
        options: {}
    });

    // Database provider options
    const dbProviders = [
        { value: 'postgresql', label: 'PostgreSQL' },
        { value: 'mysql', label: 'MySQL' },
        { value: 'mongodb', label: 'MongoDB' }
    ];

    // LLM provider options
    const llmProviders = [
        { value: 'openai', label: 'OpenAI' },
        { value: 'anthropic', label: 'Anthropic' },
        { value: 'gemini', label: 'Google Gemini' },
        { value: 'bedrock', label: 'AWS Bedrock' }
    ];

    useEffect(() => {
        fetchConfigurations();
    }, []);

    const fetchConfigurations = async () => {
        setIsLoading(true);
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
        } catch (error) {
            console.error('Failed to fetch configurations:', error);
        } finally {
            setIsLoading(false);
        }
    };

    const getProviderSpecificFields = () => {
        if (activeSection === 'database') {
            switch (formData.type) {
                case 'postgresql':
                    return (
                        <div>
                            <label className="block text-sm font-medium text-gray-700">SSL Mode</label>
                            <select
                                value={formData.options?.ssl_mode || 'disable'}
                                onChange={(e) => setFormData({
                                    ...formData,
                                    options: { ...formData.options, ssl_mode: e.target.value }
                                })}
                                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                            >
                                <option value="disable">Disable</option>
                                <option value="allow">Allow</option>
                                <option value="prefer">Prefer</option>
                                <option value="require">Require</option>
                                <option value="verify-ca">Verify CA</option>
                                <option value="verify-full">Verify Full</option>
                            </select>
                        </div>
                    );
                case 'mysql':
                    return (
                        <>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Charset</label>
                                <input
                                    type="text"
                                    value={formData.options?.charset || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        options: { ...formData.options, charset: e.target.value }
                                    })}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                    placeholder="utf8mb4"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Collation</label>
                                <input
                                    type="text"
                                    value={formData.options?.collation || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        options: { ...formData.options, collation: e.target.value }
                                    })}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                    placeholder="utf8mb4_general_ci"
                                />
                            </div>
                        </>
                    );
                case 'mongodb':
                    return (
                        <>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Auth Database</label>
                                <input
                                    type="text"
                                    value={formData.options?.auth_db || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        options: { ...formData.options, auth_db: e.target.value }
                                    })}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Replica Set</label>
                                <input
                                    type="text"
                                    value={formData.options?.replica_set || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        options: { ...formData.options, replica_set: e.target.value }
                                    })}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                />
                            </div>
                        </>
                    );
                default:
                    return null;
            }
        } else {
            switch (formData.type) {
                case 'openai':
                    return (
                        <>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Organization ID</label>
                                <input
                                    type="text"
                                    value={formData.options?.organization || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        options: { ...formData.options, organization: e.target.value }
                                    })}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Max Tokens</label>
                                <input
                                    type="number"
                                    value={formData.options?.max_tokens || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        options: { ...formData.options, max_tokens: parseInt(e.target.value) }
                                    })}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                />
                            </div>
                        </>
                    );
                // Add other LLM provider specific fields here
                default:
                    return null;
            }
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setIsLoading(true);

        try {
            const endpoint = activeSection === 'database'
                ? '/databases'
                : `/llm`;

            const payload = {
                ...formData,
                type: formData.type || (activeSection === 'database' ? 'postgresql' : 'openai')
            };

            const response = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (response.ok) {
                await fetchConfigurations();
                setShowAddForm(false);
                setFormData({
                    name: '',
                    type: '',
                    host: '',
                    port: '',
                    db_name: '',
                    username: '',
                    password: '',
                    api_key: '',
                    model: '',
                    options: {}
                });
            } else {
                const error = await response.json();
                alert(error.error || 'Failed to save configuration');
            }
        } catch (error) {
            console.error('Failed to save configuration:', error);
            alert('Failed to save configuration');
        } finally {
            setIsLoading(false);
        }
    };

    const handleDelete = async (configType, config) => {
        if (!confirm('Are you sure you want to delete this configuration?')) return;

        try {
            const endpoint = configType === 'database'
                ? `/databases/${config.name}`
                : `/llm/${config.name}`; // Updated to match API endpoint

            const response = await fetch(endpoint, {
                method: 'DELETE'
            });

            if (response.ok) {
                await fetchConfigurations();
            } else {
                const error = await response.json();
                alert(error.error || 'Failed to delete configuration');
            }
        } catch (error) {
            console.error('Failed to delete configuration:', error);
            alert('Failed to delete configuration');
        }
    };

    return (
        <div className="space-y-6">
            {/* Section Tabs */}
            <div className="border-b border-gray-200">
                <nav className="-mb-px flex space-x-8">
                    <button
                        onClick={() => setActiveSection('database')}
                        className={`
                            flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm
                            ${activeSection === 'database'
                            ? 'border-blue-500 text-blue-600'
                            : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                        }
                        `}
                    >
                        <Database className="h-4 w-4" />
                        Database Configurations
                    </button>
                    <button
                        onClick={() => setActiveSection('llm')}
                        className={`
                            flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm
                            ${activeSection === 'llm'
                            ? 'border-blue-500 text-blue-600'
                            : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                        }
                        `}
                    >
                        <Brain className="h-4 w-4" />
                        LLM Configurations
                    </button>
                </nav>
            </div>

            {/* Add Configuration Button */}
            <div className="flex justify-end">
                <button
                    onClick={() => setShowAddForm(true)}
                    className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
                >
                    <Plus className="h-4 w-4" />
                    Add Configuration
                </button>
            </div>

            {/* Configuration Form Modal */}
            {showAddForm && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
                    <div className="bg-white rounded-lg p-6 max-w-lg w-full">
                        <h3 className="text-lg font-medium mb-4">
                            Add {activeSection === 'database' ? 'Database' : 'LLM'} Configuration
                        </h3>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700">Name</label>
                                <input
                                    type="text"
                                    value={formData.name}
                                    onChange={(e) => setFormData({...formData, name: e.target.value})}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                    required
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700">Provider Type</label>
                                <select
                                    value={formData.type}
                                    onChange={(e) => setFormData({...formData, type: e.target.value})}
                                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                    required
                                >
                                    <option value="">Select Provider</option>
                                    {(activeSection === 'database' ? dbProviders : llmProviders).map(provider => (
                                        <option key={provider.value} value={provider.value}>
                                            {provider.label}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            {activeSection === 'database' ? (
                                <>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">Host</label>
                                        <input
                                            type="text"
                                            value={formData.host}
                                            onChange={(e) => setFormData({...formData, host: e.target.value})}
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                            required
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">Port</label>
                                        <input
                                            type="text"
                                            value={formData.port}
                                            onChange={(e) => setFormData({...formData, port: e.target.value})}
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                            required
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">Database Name</label>
                                        <input
                                            type="text"
                                            value={formData.db_name}
                                            onChange={(e) => setFormData({...formData, db_name: e.target.value})}
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                            required
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">Username</label>
                                        <input
                                            type="text"
                                            value={formData.username}
                                            onChange={(e) => setFormData({...formData, username: e.target.value})}
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2" required
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">Password</label>
                                        <input
                                            type="password"
                                            value={formData.password}
                                            onChange={(e) => setFormData({...formData, password: e.target.value})}
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                            required
                                        />
                                    </div>
                                </>
                            ) : (
                                <>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">API Key</label>
                                        <input
                                            type="password"
                                            value={formData.api_key}
                                            onChange={(e) => setFormData({...formData, api_key: e.target.value})}
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                            required
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700">Model</label>
                                        <input
                                            type="text"
                                            value={formData.model}
                                            onChange={(e) => setFormData({...formData, model: e.target.value})}
                                            placeholder="e.g., gpt-4"
                                            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                                            required
                                        />
                                    </div>
                                </>
                            )}

                            {/* Provider-specific options */}
                            {formData.type && getProviderSpecificFields()}

                            <div className="flex justify-end gap-3 mt-6">
                                <button
                                    type="button"
                                    onClick={() => setShowAddForm(false)}
                                    className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    disabled={isLoading}
                                    className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                                >
                                    {isLoading ? 'Saving...' : 'Save'}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {/* Configurations List */}
            <div className="bg-white shadow-sm rounded-lg border">
                <div className="px-4 py-5 sm:p-6">
                    {isLoading ? (
                        <div className="text-center text-gray-500">Loading configurations...</div>
                    ) : activeSection === 'database' ? (
                        <div className="space-y-4">
                            {configs.database.map((config) => (
                                <div key={config.name} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                                    <div>
                                        <h3 className="font-medium">{config.name}</h3>
                                        <p className="text-sm text-gray-500">
                                            {config.type} - {config.host}:{config.port} - {config.db_name}
                                        </p>
                                        {config.options && Object.keys(config.options).length > 0 && (
                                            <p className="text-xs text-gray-400 mt-1">
                                                Options: {Object.entries(config.options).map(([key, value]) =>
                                                `${key}=${value}`).join(', ')}
                                            </p>
                                        )}
                                    </div>
                                    <button
                                        onClick={() => handleDelete('database', config)}
                                        className="text-red-600 hover:text-red-700"
                                    >
                                        <Trash2 className="h-5 w-5" />
                                    </button>
                                </div>
                            ))}
                            {configs.database.length === 0 && (
                                <div className="text-center text-gray-500">No database configurations found</div>
                            )}
                        </div>
                    ) : (
                        <div className="space-y-4">
                            {configs.llm.map((config) => (
                                <div key={`${config.type}-${config.name}`} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                                    <div>
                                        <h3 className="font-medium">{config.name}</h3>
                                        <p className="text-sm text-gray-500">
                                            Provider: {config.type} - Model: {config.model}
                                        </p>
                                        {config.options && Object.keys(config.options).length > 0 && (
                                            <p className="text-xs text-gray-400 mt-1">
                                                Options: {Object.entries(config.options).map(([key, value]) =>
                                                `${key}=${value}`).join(', ')}
                                            </p>
                                        )}
                                    </div>
                                    <button
                                        onClick={() => handleDelete('llm', config)}
                                        className="text-red-600 hover:text-red-700"
                                    >
                                        <Trash2 className="h-5 w-5" />
                                    </button>
                                </div>
                            ))}
                            {configs.llm.length === 0 && (
                                <div className="text-center text-gray-500">No LLM configurations found</div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default ConfigPage;