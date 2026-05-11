import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './css/index.css'
import { createBrowserRouter, RouterProvider } from 'react-router'
import HomePage from './components/default-components/HomePage.tsx'
import Login from './components/default-components/Login.tsx'
import App from './components/App.tsx'
import StorageHome from './components/auth-components/storage-components/StorageHome.tsx'
import { authMiddleware, getUserContext } from './middleware.ts'
import Register from './components/default-components/Register.tsx'

const router = createBrowserRouter([
  {
    path: "/", 
    element: <App />,
    children: [
      {index: true, element: <HomePage />},
      {
        path: "login",
        element: <Login />
      },
      {
        path: "register",
        element: <Register />,
      },
      {
        path: "storage",
        element: <App />,
        // TODO: fix this middleware
        middleware: [authMiddleware],
        children: [
          {index: true, loader: getUserContext, element: <StorageHome />},
          {path: "folder/:folderId", loader: getUserContext, element: <StorageHome />}
        ]
      },
    ]
  },
]);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
      <RouterProvider router={router} />
  </StrictMode>,
)
