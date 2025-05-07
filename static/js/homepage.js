document.addEventListener("DOMContentLoaded", () => {
    // ========== Helper Functions ==========

    async function sendLike(endpoint, id, spanSelector) {
        const response = await fetch(endpoint, {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: `${endpoint.includes("comment") ? 'comment_id' : 'post_id'}=${id}`,
        });

        if (response.ok) {
            const updatedLikes = await response.text();
            spanSelector.textContent = updatedLikes;
        }
    }

    function toggleVisibility(element) {
        element.classList.toggle('visible');
    }

    function toggleClass(element, className) {
        element.classList.toggle(className);
    }

    function validatePostForm() {
        const tagCheckboxes = postForm.querySelectorAll('input[name="tags"]:checked');
        const contentField = postForm.querySelector('textarea[name="content"]');
        const hasTags = tagCheckboxes.length > 0;
        const hasContent = contentField.value.trim().length > 0;

        postButton.disabled = !(hasTags && hasContent);
        postButton.style.cursor = postButton.disabled ? 'not-allowed' : 'pointer';
    }

    function previewImage(fileInput, previewContainer) {
        if (fileInput.files && fileInput.files[0]) {
            const reader = new FileReader();
            reader.onload = function (e) {
                previewContainer.innerHTML = `<img src="${e.target.result}" alt="Preview" class="image-preview">`;
            };
            reader.readAsDataURL(fileInput.files[0]);
        }
    }

    // ========== Like Buttons ==========

    document.querySelectorAll(".like-btn").forEach(button => {
        button.addEventListener("click", async () => {
            const postID = button.dataset.postId;
            const likeCount = button.querySelector(".like-count");
            await sendLike("/like", postID, likeCount);
        });
    });

    document.querySelectorAll(".comment-like-btn").forEach(button => {
        button.addEventListener("click", async (e) => {
            e.preventDefault();
            const commentID = button.dataset.commentId;
            const likeCount = button.querySelector(".comment-like-count");
            await sendLike("/like-comment", commentID, likeCount);
        });
    });

    // ========== Comment Toggle ==========

    document.querySelectorAll(".toggle-comments-btn").forEach(button => {
        button.addEventListener("click", function (e) {
            e.preventDefault();
            const comments = this.closest('.post').querySelector('.comments');
            toggleVisibility(comments);
            toggleClass(this, 'active');
        });
    });

    // ========== Post Form Validation ==========

    const postForm = document.querySelector('.post-form form');
    const postButton = postForm?.querySelector('button[type="submit"]');

    if (postForm && postButton) {
        validatePostForm();

        postForm.querySelectorAll('input[name="tags"]').forEach(cb => {
            cb.addEventListener('change', validatePostForm);
        });

        postForm.querySelector('textarea[name="content"]').addEventListener('input', validatePostForm);
    }

    // ========== Image Preview ==========

    const imageInput = document.getElementById('post-image');
    const imagePreview = document.getElementById('image-preview');

    if (imageInput && imagePreview) {
        imageInput.addEventListener('change', function () {
            previewImage(this, imagePreview);
        });
    }
});
