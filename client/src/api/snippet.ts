interface SnippetResponse {
	ID: string;
	Language: string;
	Snippet: string[]; // it comes as a JSON string in your response
}
export async function getsnippet(lang: string): Promise<SnippetResponse> {
	try {
		const res = await fetch(`http://localhost:8080/get-snippet?lang=${lang}`);
		const data: SnippetResponse = await res.json();
		console.log(data);
		return data;
	} catch (err) {
		console.error(err);
		throw err;
	}
}
