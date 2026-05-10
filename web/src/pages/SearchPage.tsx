import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import Nav from '../components/Nav'
import { usePropertyDocumentsManagerCertificatesUpload } from '../api/koditon'

const targetTypes = [
  ['offering', 'Offering'],
  ['unit', 'Unit'],
  ['building', 'Building'],
  ['housing_company', 'Housing company'],
] as const

export default function SearchPage() {
  const navigate = useNavigate()
  const uploadMutation = usePropertyDocumentsManagerCertificatesUpload()
  const [targetType, setTargetType] = useState('offering')
  const [targetID, setTargetID] = useState('')
  const [uploadTargetType, setUploadTargetType] = useState('offering')
  const [uploadTargetID, setUploadTargetID] = useState('')
  const [uploadMessage, setUploadMessage] = useState('')
  function openTarget(event: FormEvent) {
    event.preventDefault()
    if (!targetID.trim()) return
    navigate(`/target/${targetType}/${targetID.trim()}`)
  }
  async function uploadCertificate(file: File | undefined) {
    if (!file) return
    setUploadMessage('')
    const params = uploadTargetID.trim() ? { target_type: uploadTargetType, target_id: uploadTargetID.trim() } : undefined
    const response = await uploadMutation.mutateAsync({ data: { file }, params })
    const body = response.data as { document?: { id?: string } }
    const documentID = body.document?.id ?? 'document'
    setUploadMessage(`Uploaded ${documentID}`)
    if (params) {
      navigate(`/target/${uploadTargetType}/${uploadTargetID.trim()}`)
    }
  }
  return (
    <main className="model-page">
      <Nav />
      <div className="model-shell model-shell--narrow">
        <header className="model-header">
          <div>
            <h1>Property model</h1>
            <p>Open canonical targets, inspect resolved values, and attach source documents.</p>
          </div>
        </header>
        <div className="model-start-grid">
          <section className="model-panel">
            <header>
              <h2>Open Target</h2>
            </header>
            <form className="model-form" onSubmit={openTarget}>
              <label>
                <span>Target type</span>
                <select value={targetType} onChange={event => setTargetType(event.target.value)}>
                  {targetTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                </select>
              </label>
              <label>
                <span>Target id</span>
                <input value={targetID} onChange={event => setTargetID(event.target.value)} placeholder="UUID" />
              </label>
              <button type="submit">Open target</button>
            </form>
          </section>
          <section className="model-panel">
            <header>
              <h2>Upload Certificate</h2>
            </header>
            <div className="model-form">
              <label>
                <span>Attach to</span>
                <select value={uploadTargetType} onChange={event => setUploadTargetType(event.target.value)}>
                  {targetTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                </select>
              </label>
              <label>
                <span>Target id</span>
                <input value={uploadTargetID} onChange={event => setUploadTargetID(event.target.value)} placeholder="Leave empty for detached upload" />
              </label>
              <label className="model-file-button">
                {uploadMutation.isPending ? 'Uploading...' : 'Choose PDF'}
                <input disabled={uploadMutation.isPending} type="file" accept="application/pdf" onChange={event => uploadCertificate(event.target.files?.[0])} />
              </label>
              {uploadMessage && <p className="model-form-note">{uploadMessage}</p>}
              {uploadMutation.isError && <p className="model-form-error">Upload failed.</p>}
            </div>
          </section>
        </div>
      </div>
    </main>
  )
}
