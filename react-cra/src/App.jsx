import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

// Payment method components.
import Ideal from './Ideal';
import IssueCredential from './IssueCredential';

import './App.css';

function App(props) {
  return (
    <>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Ideal />} />
          <Route path="/return" element={<IssueCredential />} />
        </Routes>
      </BrowserRouter>

    </>
  );
}

export default App;
