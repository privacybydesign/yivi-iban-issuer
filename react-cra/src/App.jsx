import React, { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import Ideal from './Ideal';
import IssueCredential from './IssueCredential';

import './App.css';
import './i18n';

// Wrapper that sets the language based on the URL
function LanguageRouter() {
  const { lang } = useParams();
  const { i18n } = useTranslation();

  useEffect(() => {
    if (lang && i18n.language !== lang) {
      i18n.changeLanguage(lang);
    }
  }, [lang, i18n]);

  return (
    <Routes>
      <Route path="/" element={<Ideal />} />
      <Route path="/return" element={<IssueCredential />} />
    </Routes>
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Redirect base URL to default language (en) */}
        <Route path="/" element={<Navigate to="/en" replace />} />

        {/* Route language-prefixed URLs like /en, /nl, etc. */}
        <Route path=":lang/*" element={<LanguageRouter />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;