import React from 'react'
import {createRoot} from 'react-dom/client'
import {ConfigProvider} from '@douyinfe/semi-ui'
import zh_CN from '@douyinfe/semi-ui/lib/es/locale/source/zh_CN'
import './style.css'
import App from './App'

document.body.setAttribute('theme-mode', 'dark')

const container = document.getElementById('root')
const root = createRoot(container!)
root.render(
    <React.StrictMode>
        <ConfigProvider locale={zh_CN}>
            <App/>
        </ConfigProvider>
    </React.StrictMode>
)
