import { useState } from 'react'
import Layout from './components/Layout'
import SmartInsights from './components/SmartInsights'
import ConfigPage from './components/ConfigPage'

function App() {
    const [currentTab, setCurrentTab] = useState('chat')

    return (
        <Layout currentTab={currentTab} setCurrentTab={setCurrentTab}>
            {currentTab === 'chat' ? <SmartInsights /> : <ConfigPage />}
        </Layout>
    )
}

export default App