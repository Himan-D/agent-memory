import os
import json
import urllib.request
import urllib.error

def evaluate_compression_fidelity(original_text, compressed_output) -> dict:
    """
    Library-agnostic accuracy evaluator using Gemini's native API structure.
    Bypasses OpenAI translation logic entirely to fix authorization schema bugs.
    """
    api_key = os.environ.get("EVALUATOR_API_KEY", "").strip()
    base_url = os.environ.get("EVALUATOR_BASE_URL", "https://generativelanguage.googleapis.com/v1beta").rstrip("/")
    model = os.environ.get("EVALUATOR_MODEL", "gemini-2.0-flash")
    
    prompt = f"""
    You are an expert data QA engineer evaluating an AI memory compression engine.
    
    ORIGINAL TEXT:
    \"\"\"{original_text}\"\"\"
    
    COMPRESSED OUTPUT (FACTS/GRAPH):
    \"\"\"{compressed_output}\"\"\"
    
    Analyze the compressed output against the original text and calculate two scores between 0.0 and 1.0:
    1. Recall (Factual Retention): What percentage of the critical information from the original text was successfully preserved? (1.0 = everything preserved, 0.0 = completely lost).
    2. Precision (No Hallucination): Did the compression introduce any false assumptions or hallucinations not present in the original text? (1.0 = completely clean, 0.0 = completely hallucinated).
    
    Respond STRICTLY in the following raw JSON format:
    {{
        "recall": 0.95,
        "precision": 1.0,
        "reasoning": "Brief narrative explanation."
    }}
    """
    
    # Native Gemini API structure path
    url = f"{base_url}/models/{model}:generateContent"
    
    payload = {
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {
            "responseMimeType": "application/json",
            "temperature": 0.0
        }
    }
    
    headers = {
        "Content-Type": "application/json",
        "x-goog-api-key": api_key  # Native Google API key injection header
    }

    try:
        data = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(url, data=data, headers=headers, method="POST")
        
        with urllib.request.urlopen(req, timeout=30) as response:
            res_body = response.read().decode("utf-8")
            res_json = json.loads(res_body)
            
            # Extract textual choice payload response from native Google structure
            text_response = res_json["candidates"][0]["content"]["parts"][0]["text"]
            return json.loads(text_response)
            
    except urllib.error.HTTPError as e:
        err_details = e.read().decode('utf-8', errors='ignore')
        return {"recall": 0.0, "precision": 0.0, "error": f"HTTP {e.code}: {err_details}"}
    except Exception as e:
        return {"recall": 0.0, "precision": 0.0, "error": str(e)}