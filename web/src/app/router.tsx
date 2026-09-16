import type { RouteObject } from 'react-router-dom'

import { Providers } from '@/app/providers'
import { RootErrorBoundary } from '@/app/root-error-boundary'
import {
  CallbackPage,
  ForgotPasswordPage,
  ProtectedLayout,
  ResetPasswordPage,
  SignInPage,
  SignUpPage,
} from '@/features/auth'
import {
  AgreementsListPage,
  AgreementWizardPage,
  OrganizationPage,
} from '@/features/agreements'
import { DashboardPage } from '@/features/dashboard'
import { NotFoundPage } from '@/pages/not-found-page'

// The app's whole page inventory lives in this file. Authenticated
// routes nest under ProtectedLayout.
export const routes: RouteObject[] = [
  {
    element: <Providers />,
    errorElement: <RootErrorBoundary />,
    children: [
      { path: '/', element: <SignInPage /> },
      { path: '/sign-up', element: <SignUpPage /> },
      { path: '/forgot-password', element: <ForgotPasswordPage /> },
      { path: '/reset-password', element: <ResetPasswordPage /> },
      { path: '/callback', element: <CallbackPage /> },
      {
        element: <ProtectedLayout />,
        children: [
          { path: '/dashboard', element: <DashboardPage /> },
          { path: '/agreements', element: <AgreementsListPage /> },
          { path: '/agreements/new', element: <AgreementWizardPage /> },
          { path: '/agreements/:id/edit', element: <AgreementWizardPage /> },
          { path: '/organization', element: <OrganizationPage /> },
        ],
      },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
]
