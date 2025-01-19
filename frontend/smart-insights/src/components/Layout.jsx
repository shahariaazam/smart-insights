import React from 'react';
import {BarChart3, Database, MessageSquare, Settings} from 'lucide-react';

const Layout = ({ currentTab, setCurrentTab, children }) => {
    const tabs = [
        { id: 'chat', label: 'Chat', icon: MessageSquare },
        { id: 'config', label: 'Configurations', icon: Settings },
    ];

    return (
        <div className="min-h-screen bg-gray-50">
            <header className="bg-white border-b">
                <div className="max-w-7xl mx-auto px-4 py-6">
                    <div className="flex items-center gap-3">
                        <BarChart3 className="h-8 w-8 text-blue-600" />
                        <h1 className="text-2xl font-semibold text-gray-900">Smart Insights</h1>
                    </div>
                    <p className="mt-1 text-sm text-gray-500">Effortless data insights, intelligently delivered</p>
                </div>
                <div className="max-w-7xl mx-auto px-4">
                    <div className="border-b border-gray-200">
                        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                            {tabs.map((tab) => {
                                const Icon = tab.icon;
                                return (
                                    <button
                                        key={tab.id}
                                        onClick={() => setCurrentTab(tab.id)}
                                        className={`
                                            flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm
                                            ${currentTab === tab.id
                                            ? 'border-blue-500 text-blue-600'
                                            : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                                        }
                                        `}
                                    >
                                        <Icon className="h-4 w-4" />
                                        {tab.label}
                                    </button>
                                );
                            })}
                        </nav>
                    </div>
                </div>
            </header>
            <main className="max-w-7xl mx-auto px-4 py-8">
                {children}
            </main>
        </div>
    );
};

export default Layout;