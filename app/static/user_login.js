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
      sessionStorage.setItem("flyteam_user_csrf", data.csrf_token || "");
      msg.textContent = "登录成功";
      window.location.href = "/";
    } catch (err) {
      msg.textContent = err.message || "登录失败";
    }
  });
}
