const form = document.getElementById("userRegisterForm");
const msg = document.getElementById("userRegisterMsg");

if (form && msg) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const nickname = document.getElementById("nickname").value.trim();
    const userId = document.getElementById("userId").value.trim();
    const password = document.getElementById("password").value;
    msg.textContent = "注册中...";
    try {
      const res = await fetch("/api/users/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ nickname, user_id: userId, password }),
        credentials: "same-origin",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.detail || "注册失败");
      msg.textContent = "注册成功，请登录。";
      setTimeout(() => {
        window.location.href = "/user-login";
      }, 700);
    } catch (err) {
      msg.textContent = err.message || "注册失败";
    }
  });
}
