const form = document.getElementById("userLoginForm");
const msg = document.getElementById("userLoginMsg");

if (form && msg) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const userId = document.getElementById("userId").value.trim();
    const password = document.getElementById("password").value;
    msg.textContent = "登录中...";
    try {
      const res = await fetch("/api/users/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, password }),
        credentials: "same-origin",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.detail || "登录失败");
      localStorage.setItem("flyteam_user_token", data.token || "");
      localStorage.setItem("user_token", data.token || "");
      sessionStorage.setItem("flyteam_user_csrf", data.csrf_token || "");
      if (data.user) localStorage.setItem("flyteam_user", JSON.stringify(data.user));
      msg.textContent = "登录成功";
      const next = new URLSearchParams(location.search).get("next") || "/blog";
      window.location.href = next.startsWith("/") ? next : "/blog";
    } catch (err) {
      msg.textContent = err.message || "登录失败";
    }
  });
}
